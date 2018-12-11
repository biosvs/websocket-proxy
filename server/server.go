package server

import (
	"net/http"

	"websocket-proxy/config"
	"websocket-proxy/connection"
	"websocket-proxy/hub"
	"websocket-proxy/proxy"

	log "github.com/sirupsen/logrus"
)

// Server represents server handler for incoming requests
type Server struct {
	h         *hub.Hub
	srvConfig *config.ServerConfig
	dstConfig *config.DstConfig
}

// New create new server
func New(h *hub.Hub, srvConfig *config.ServerConfig, dstConfig *config.DstConfig) *Server {
	return &Server{
		h, srvConfig, dstConfig,
	}
}

// Start run handlers
func (s *Server) Start() {
	if s.srvConfig.Path != "/" {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello! This is blank page, well, you can set your cookies, for example."))
		})
	}

	http.HandleFunc(s.srvConfig.Path, func(w http.ResponseWriter, r *http.Request) {
		s.handler(w, r)
	})

	log.Infof("Starting proxy server on %s:%s%s to %s", s.srvConfig.Host, s.srvConfig.Port, s.srvConfig.Path, proxy.GetDstURL(s.dstConfig))
	if err := http.ListenAndServe(s.srvConfig.Host+":"+s.srvConfig.Port, nil); err != nil {
		log.Fatal("Cannot start server: ", err)
	}
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		log.Info("server handler: got bad method: ", r.Method)
		w.Write([]byte("GET method only acceptable"))
		return
	}

	con := connection.New(s.h.GetNextName())
	if err := s.h.AddConnection(con); err != nil {
		log.Infof("server handler: cannot add connection: %v", err)
		w.Write([]byte("System error, try later (1)"))
		return
	}

	if err := con.Init(w, r, s.dstConfig); err != nil {
		s.h.DropConnection(con.GetUniq())
		log.Infof("server handler: cannot init connection: %v", err)
		w.Write([]byte("System error, try later (2)"))
		return
	}
}
