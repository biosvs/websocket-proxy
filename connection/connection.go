package connection

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"

	"websocket-proxy/config"
	"websocket-proxy/proxy"

	log "github.com/sirupsen/logrus"
)

const (
	stateInit = iota
	stateRun
	stateStopped
)

// Connection represents client connection and proxy for this connection
type Connection struct {
	sync.RWMutex
	state  int
	uniq   int
	name   int
	proxy  *proxy.Proxy
	stopCh chan struct{}
}

// New create new connection
func New(name int) *Connection {
	return &Connection{
		state: stateInit,
		uniq:  rand.Int(),
		name:  name,
	}
}

// GetUniq get uniq name of current connection
func (c *Connection) GetUniq() int {
	return c.uniq
}

// Init initialize connection
func (c *Connection) Init(w http.ResponseWriter, r *http.Request, conf *config.DstConfig) error {
	c.Lock()
	defer c.Unlock()

	if c.state != stateInit {
		return fmt.Errorf("connection %d init: bad state", c.name)
	}

	c.proxy = proxy.New(c.name)
	go func() {
		select {
		case err := <-c.proxy.ErrCh:
			log.Infof("connection %d stop: proxy err: %v", c.name, err)
			c.Close()

		case <-c.stopCh:
			log.Infof("connection %d stop: by stop event", c.name)
		}
	}()

	if err := c.proxy.Run(w, r, conf); err != nil {
		return fmt.Errorf("connection %d init: proxy error: %v", c.name, err)
	}

	c.state = stateRun
	return nil
}

// Close close connection gracefully
func (c *Connection) Close() {
	c.Lock()
	if c.state == stateInit {
		c.state = stateStopped
		c.Unlock()
		return
	}

	c.proxy.Stop()
	c.stopCh <- struct{}{}

	c.state = stateStopped
	c.Unlock()
}
