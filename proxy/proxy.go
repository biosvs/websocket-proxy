package proxy

import (
	"fmt"
	"io"
	"net/http"
	"sync"

	"websocket-proxy/config"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Proxy represents logic of proxy between client and server
type Proxy struct {
	internalStopCh chan struct{}
	ErrCh          chan error

	name         int
	clientWsConn *websocket.Conn
	serverWsConn *websocket.Conn
}

// New create new proxy
func New(name int) *Proxy {
	return &Proxy{
		internalStopCh: make(chan struct{}),
		ErrCh:          make(chan error),
		name:           name,
	}
}

// Run run ws-proxy
func (p *Proxy) Run(w http.ResponseWriter, r *http.Request, c *config.DstConfig) error {
	var err error
	dstURL := GetDstURL(c)

	p.clientWsConn, err = upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Infof("proxy %d: upgrade socket error: %v", p.name, err)
		return err
	}

	// Connect to dst server
	header := http.Header{}
	for _, cookie := range r.Header[http.CanonicalHeaderKey("Cookie")] {
		log.Infof("proxy %d: cookie: %v", p.name, cookie)
		header.Add("Cookie", cookie)
	}

	var resp *http.Response
	p.serverWsConn, resp, err = websocket.DefaultDialer.Dial(dstURL, header)
	if err != nil {
		log.Warnf("proxy %d: could not connect to %s: %s", p.name, dstURL, err)
		if resp != nil {
			if err := copyResponse(w, resp); err != nil {
				log.Warnf("proxy %d: could not get response at all: %s", p.name, err)
				return err
			}
		} else {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		}
		return err
	}

	log.Infof("proxy %d: connection opened", p.name)

	go p.runReplicator()

	return nil
}

func (p *Proxy) runReplicator() {
	errClient := make(chan error, 1)
	errServer := make(chan error, 1)
	stopClient := make(chan struct{}, 1)
	stopServer := make(chan struct{}, 1)

	wg := sync.WaitGroup{}
	wg.Add(2)

	replicateWebsocketConn := func(src, dst *websocket.Conn, errCh chan error, stopCh chan struct{}, server bool) {
		defer wg.Done()

		socketCh := make(chan socketReadResult)
		unlockRead := make(chan struct{})

		go func() {
			for {
				msgType, msg, err := src.ReadMessage()
				socketCh <- socketReadResult{msgType, msg, err}
				<-unlockRead
			}
		}()

		for {
			select {
			case <-stopCh:
				m := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Interrupted")
				dst.WriteMessage(websocket.CloseMessage, m)
				log.Infof("%s: connection closed (interrupted)", p.formatReplicaName(server))
				return

			case socketMsg := <-socketCh:
				msgType := socketMsg.msgType
				msg := socketMsg.msg
				err := socketMsg.err

				if err != nil {
					m := websocket.FormatCloseMessage(websocket.CloseNormalClosure, fmt.Sprintf("%v", err))
					if e, ok := err.(*websocket.CloseError); ok {
						if e.Code != websocket.CloseNoStatusReceived {
							m = websocket.FormatCloseMessage(e.Code, e.Text)
							log.Infof("%s: connection closed", p.formatReplicaName(server))
						}
					}
					dst.WriteMessage(websocket.CloseMessage, m)
					errCh <- err
					return
				}

				// log message
				var txtMsg string
				if msgType == websocket.TextMessage {
					txtMsg = string(msg)
				}

				log.Infof("%s: %s", p.formatReplicaName(server), txtMsg)

				err = dst.WriteMessage(msgType, msg)
				if err != nil {
					log.Infof("%s: cannot write message: %v", p.formatReplicaName(server), err)
					errCh <- err
					return
				}

				unlockRead <- struct{}{}
			}
		}
	}

	go replicateWebsocketConn(p.serverWsConn, p.clientWsConn, errClient, stopClient, true)
	go replicateWebsocketConn(p.clientWsConn, p.serverWsConn, errServer, stopServer, false)

	var (
		message string
		err     error
	)
	select {
	case err = <-errClient:
		message = "Error when copying from server to client: %v"
		p.ErrCh <- err
	case err = <-errServer:
		message = "Error when copying from client to server: %v"
		p.ErrCh <- err
	case <-p.internalStopCh:
		log.Infof("proxy %d: got signal: interrupted", p.name)
		stopServer <- struct{}{}
		stopClient <- struct{}{}
	}
	if message != "" {
		if e, ok := err.(*websocket.CloseError); !ok || e.Code == websocket.CloseAbnormalClosure {
			log.Infof("proxy %d: %s %v", p.name, message, err)
		}
	}

	wg.Wait()
	p.closeTCP()
}

// Stop stop proxy gracefully
func (p *Proxy) Stop() {
	p.internalStopCh <- struct{}{}
}

func (p *Proxy) closeTCP() {
	p.clientWsConn.Close()
	p.serverWsConn.Close()
}

func (p *Proxy) formatReplicaName(server bool) string {
	srcName := "client to server"
	if server {
		srcName = "server to client"
	}
	return fmt.Sprintf("proxy %d (%s)", p.name, srcName)
}

type socketReadResult struct {
	msgType int
	msg     []byte
	err     error
}

// GetDstURL return destination host url
func GetDstURL(c *config.DstConfig) string {
	scheme := "wss"
	if c.Insecure {
		scheme = "ws"
	}

	if c.Port != "" {
		return fmt.Sprintf("%s://%s:%s%s", scheme, c.Host, c.Port, c.Path)
	}
	return fmt.Sprintf("%s://%s%s", scheme, c.Host, c.Path)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func copyResponse(rw http.ResponseWriter, resp *http.Response) error {
	copyHeader(rw.Header(), resp.Header)
	rw.WriteHeader(resp.StatusCode)
	defer resp.Body.Close()

	_, err := io.Copy(rw, resp.Body)
	return err
}
