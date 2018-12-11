package config

import "strings"

// ServerConfig config of server
type ServerConfig struct {
	Host, Port, Path string
}

// DstConfig config of destination host
type DstConfig struct {
	ServerConfig
	Insecure bool
}

// Validate validate input
func (sc *ServerConfig) Validate() {
	sc.Path = "/" + strings.TrimLeft(sc.Path, "/")
}
