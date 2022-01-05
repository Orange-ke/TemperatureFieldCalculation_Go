package server

import (
	"flag"
	"log"
	"lz/model"
	"net/http"

	"github.com/gorilla/websocket"
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
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()
	var msg model.Msg
	for {
		err = conn.ReadJSON(&msg)
		if err != nil {
			log.Println("err: ", err)
		}
		switch msg.Type {
		case "env":
			reply := model.Msg{
				Type: "envSet",
				Content: "env is set",
			}
			err = conn.WriteJSON(&reply)
			if err != nil {
				log.Println("err: ", err)
			}
		case "start":
			reply := model.Msg{
				Type: "started",
				Content: "started",
			}
			err = conn.WriteJSON(&reply)
			if err != nil {
				log.Println("err: ", err)
			}
		case "stop":
			reply := model.Msg{
				Type: "stopped",
				Content: "stopped",
			}
			err = conn.WriteJSON(&reply)
			if err != nil {
				log.Println("err: ", err)
			}
		default:
			log.Println("no such type")
		}
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
