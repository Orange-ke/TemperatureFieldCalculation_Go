package server

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"lz/calculator"
	"lz/model"
	"time"
)

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	c    *calculator.Calculator
	conn *websocket.Conn
	// request
	msg chan model.Msg
	// response
	envSet  chan struct{}
	started chan struct{}
	stopped chan struct{}
}

func NewHub() *Hub {
	return &Hub{
		msg:     make(chan model.Msg, 10),
		envSet:  make(chan struct{}, 10),
		started: make(chan struct{}, 10),
		stopped: make(chan struct{}, 10),
	}
}

func (h *Hub) handleResponse() {
	for {
		select {
		case <-h.envSet:
			reply := model.Msg{
				Type:    "envSet",
				Content: "env is set",
				// 可以加一些需要的参数过去
			}
			err := h.conn.WriteJSON(&reply)
			if err != nil {
				log.Println("err: ", err)
			}
		case <-h.started:
			// 从calculator里面的hub中获取是否有
			reply := model.Msg{
				Type:    "started",
				Content: "Started",
			}
			err := h.conn.WriteJSON(&reply)
			if err != nil {
				log.Println("err: ", err)
			}
			h.c.CalcHub.StartSignal()
			go h.c.Run()
			go h.pushData()
		case <-h.stopped:
			h.c.CalcHub.StopSignal()
			reply := model.Msg{
				Type:    "stopped",
				Content: "stopped",
			}
			err := h.conn.WriteJSON(&reply)
			if err != nil {
				log.Println("err: ", err)
			}
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (h *Hub) handleRequest() {
	for {
		select {
		case msg := <-h.msg:
			switch msg.Type {
			case "env":
				h.envSet <- struct{}{}
			case "start":
				h.started <- struct{}{}
			case "stop":

				h.stopped <- struct{}{}
			default:
				log.Println("no such type")
			}
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (h *Hub) pushData() {
	reply := model.Msg{
		Type: "data_push",
	}
Loop:
	for {
		select {
		case <-h.c.CalcHub.Stop:
			break Loop
		case <-h.c.CalcHub.PeriodCalcResult:
			temperatureData := h.c.BuildData()
			data, err := json.Marshal(temperatureData)
			if err != nil {
				log.Println("err: ", err)
			}
			reply.Content = string(data)
			err = h.conn.WriteJSON(&reply)
			if err != nil {
				log.Println("err: ", err)
			}
		}
	}
}
