package main

import (
	"github.com/gorilla/websocket"
	"lz/server"
	"net/http"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	s := server.NewServer(":9000", upgrader)
	s.Serve()
}
