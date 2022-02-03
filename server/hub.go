package server

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"lz/calculator"
	"lz/model"
	"time"
)

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	c    calculator.Calculator
	conn *websocket.Conn
	// request
	msg chan model.Msg
	// response
	envSet    chan model.Env
	started   chan struct{}
	stopped   chan struct{}
	tailStart chan struct{} // 拉尾坯
}

func NewHub() *Hub {
	return &Hub{
		msg:       make(chan model.Msg, 10),
		envSet:    make(chan model.Env, 10),
		started:   make(chan struct{}, 10),
		stopped:   make(chan struct{}, 10),
		tailStart: make(chan struct{}, 10),
	}
}

func (h *Hub) handleResponse() {
	for {
		select {
		case env := <-h.envSet: // 设置计算环境
			if h.c == nil {
				//c := calculator.NewCalculator(0)
				//c := calculator.NewCalculatorWithListDeque(0)
				h.c = calculator.NewCalculatorWithArrDeque(0)
			}
			h.c.SetCoolerConfig(env)          // 设置冷却参数
			h.c.SetV(env.DragSpeed)           // 设置拉速
			h.c.InitParameter(env.SteelValue) // 设置钢种物性参数
			reply := model.Msg{
				Type:    "env_set",
				Content: "env is set",
			}
			err := h.conn.WriteJSON(&reply)
			if err != nil {
				log.Println("err: ", err)
			}
		case <-h.started: // 开始计算
			// 从calculator里面的hub中获取是否有
			h.c.GetCalcHub().StartSignal()
			reply := model.Msg{
				Type:    "started",
				Content: "Started",
			}
			err := h.conn.WriteJSON(&reply)
			if err != nil {
				log.Println("err: ", err)
			}
			go h.c.Run()    // 不断计算
			go h.pushData() // 获取推送的计算结果到前端
		case <-h.stopped: // 停止计算
			h.c.GetCalcHub().StopSignal()
			reply := model.Msg{
				Type:    "stopped",
				Content: "stopped",
			}
			err := h.conn.WriteJSON(&reply)
			if err != nil {
				log.Println("err: ", err)
			}
		case <-h.tailStart: // 拉尾坯
			h.c.SetStateTail()
			reply := model.Msg{
				Type:    "tail_start",
				Content: "started to tail",
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
	// 可以在此对请求进行预处理
	for {
		select {
		case msg := <-h.msg:
			switch msg.Type {
			case "env":
				var env model.Env
				err := json.Unmarshal([]byte(msg.Content), &env)
				if err != nil {
					log.Println("err: ", err)
				}
				fmt.Println("获取到env参数：", env)
				h.envSet <- env
			case "start":
				h.started <- struct{}{}
			case "stop":
				h.stopped <- struct{}{}
			case "tail":
				h.tailStart <- struct{}{}
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
		case <-h.c.GetCalcHub().Stop:
			break Loop
		case <-h.c.GetCalcHub().PeriodCalcResult:
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
