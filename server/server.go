package server

import (
	"flag"
	"github.com/gorilla/websocket"
	"log"
	"lz/model"
	"net/http"
)

type Server struct {
	addr     string
	upgrader websocket.Upgrader
}

func NewServer(addr string, upgrader websocket.Upgrader) *Server {
	return &Server{
		addr:     addr,
		upgrader: upgrader,
	}
}

// serveWs handles websocket requests from the peer.
func (s *Server) serveWs(w http.ResponseWriter, r *http.Request) {
	hub := NewHub()
	conn, err := s.upgrader.Upgrade(w, r, nil)
	hub.conn = conn
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()
	var msg model.Msg
	go hub.handleRequest()
	go hub.handleResponse()
	for {
		err = conn.ReadJSON(&msg)
		if err != nil {
			log.Println("err: ", err)
			continue
		}
		hub.msg <- msg
	}
}

func (s *Server) Serve() {
	flag.Parse()
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		s.serveWs(w, r)
	})
	err := http.ListenAndServe(s.addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
