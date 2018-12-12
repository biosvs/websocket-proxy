# WebSocket Proxy

This is proxy for debugging any websocket interactions.

### How to build
[Go](http://golang.org/) version 1.11+ required.

    git clone https://github.com/biosvs/websocket-proxy.git && cd websocket-proxy && go build -o websocket-proxy
    
### How to use
    $ ./websocket-proxy --help
    -dst-host string
      	Destination host (default "127.0.0.1")
    -dst-path string
      	Destination path
    -dst-port string
      	Destination port
    -host string
      	Run on host
    -insecure
      	Use ws instead of wss for destination connect
    -log-file string
      	Path to log file (default "/tmp/websocket-proxy.log")
    -path string
      	URI path
    -port string
      	Run on port

Example:

    ./websocket-proxy --port 4041 --path ws-proxy --dst-host example.com --dst-path websock
    
Result (in log):

    Starting proxy server on :4041/ws-proxy to wss://example.com/websock
    

### Based on

[Gorilla WebSocket](https://github.com/gorilla/websocket)

[websocketproxy, Koding, Inc.](https://github.com/koding/websocketproxy)
