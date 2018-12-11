package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"
	"websocket-proxy/config"
	"websocket-proxy/hub"
	"websocket-proxy/server"

	log "github.com/sirupsen/logrus"
)

/**
websocket-proxy --port 4041 --path ws-proxy --dst-host 127.0.0.1 --dst-port 4040 --dst-path wsp-test
*/
var (
	flagLogFile     = flag.String("log-file", "/tmp/websocket-proxy.log", "Path to log file")
	flagHost        = flag.String("host", "", "Run on host")
	flagPort        = flag.String("port", "", "Run on port")
	flagPath        = flag.String("path", "", "URI path")
	flagDstHost     = flag.String("dst-host", "127.0.0.1", "Destination host")
	flagDstPort     = flag.String("dst-port", "", "Destination port")
	flagDstPath     = flag.String("dst-path", "", "Destination path")
	flagDstInsecure = flag.Bool("insecure", false, "Use ws instead of wss for destination connect")
)

func main() {
	flag.Parse()

	if *flagPort == "" {
		log.Fatal("You have to specify port")
	}

	initLogs(*flagLogFile)

	serverConfig := &config.ServerConfig{
		Host: *flagHost,
		Port: *flagPort,
		Path: *flagPath,
	}

	dstConfig := &config.DstConfig{
		ServerConfig: config.ServerConfig{
			Host: *flagDstHost,
			Port: *flagDstPort,
			Path: *flagDstPath,
		},
		Insecure: *flagDstInsecure,
	}

	serverConfig.Validate()
	dstConfig.Validate()

	h := hub.New()

	// handle os signals here
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT)
	go func() {
		<-signalCh
		go h.CloseAllConnections()
		<-time.After(500 * time.Millisecond)
		os.Exit(0)
	}()

	server.New(h, serverConfig, dstConfig).Start()
}

func initLogs(file string) {
	logFile, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0660)
	if err != nil {
		log.Fatalf("Failed to init logs: %v", err)
	}
	log.SetOutput(logFile)
	log.SetFormatter(&log.JSONFormatter{})
}
