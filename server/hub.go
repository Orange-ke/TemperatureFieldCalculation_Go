package server

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"lz/calculator"
	"lz/model"
	"strconv"
	"sync"
	"time"
)

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	c    calculator.Calculator
	conn *websocket.Conn
	// request
	msg chan model.Msg
	// response
	envSet                chan model.Env
	changeInitialTemp     chan float32
	changeNarrowSurface   chan model.NarrowSurface
	changeWideSurface     chan model.WideSurface
	changeSprayTemp       chan float32
	changeRollerWaterTemp chan float32
	changeV               chan float32
	started               chan struct{}
	stopped               chan struct{}
	tailStart             chan struct{} // 拉尾坯
	startPushSliceDetail  chan int
	stopPushSliceDetail   chan struct{}

	mu sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		msg:                   make(chan model.Msg, 10),
		envSet:                make(chan model.Env, 10),
		changeInitialTemp:     make(chan float32, 10),
		changeNarrowSurface:   make(chan model.NarrowSurface, 10),
		changeWideSurface:     make(chan model.WideSurface, 10),
		changeSprayTemp:       make(chan float32, 10),
		changeRollerWaterTemp: make(chan float32, 10),
		changeV:               make(chan float32, 10),
		started:               make(chan struct{}, 10),
		stopped:               make(chan struct{}, 10),
		tailStart:             make(chan struct{}, 10),
		startPushSliceDetail:  make(chan int, 10),
		stopPushSliceDetail:   make(chan struct{}, 10),
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
			h.mu.Lock()
			err := h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.Println("err: ", err)
			}
		case temp := <-h.changeInitialTemp:
			h.c.SetStartTemperature(temp)
			reply := model.Msg{
				Type:    "initial_temp_set",
				Content: "initial_temp_set",
			}
			h.mu.Lock()
			err := h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.Println("err: ", err)
			}
		case narrowSurface := <-h.changeNarrowSurface:
			h.c.SetNarrowSurfaceIn(narrowSurface.In)
			h.c.SetNarrowSurfaceOut(narrowSurface.Out)
			reply := model.Msg{
				Type:    "narrow_surface_temp_set",
				Content: "narrow_surface_temp_set",
			}
			h.mu.Lock()
			err := h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.Println("err: ", err)
			}
		case wideSurface := <-h.changeWideSurface:
			h.c.SetWideSurfaceIn(wideSurface.In)
			h.c.SetWideSurfaceOut(wideSurface.Out)
			reply := model.Msg{
				Type:    "wide_surface_temp_set",
				Content: "wide_surface_temp_set",
			}
			h.mu.Lock()
			err := h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.Println("err: ", err)
			}
		case temp := <-h.changeSprayTemp:
			h.c.SetSprayTemperature(temp)
			reply := model.Msg{
				Type:    "spray_water_temp_set",
				Content: "spray_water_temp_set",
			}
			h.mu.Lock()
			err := h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.Println("err: ", err)
			}
		case temp := <-h.changeRollerWaterTemp:
			h.c.SetRollerWaterTemperature(temp)
			reply := model.Msg{
				Type:    "roller_water_temp_set",
				Content: "roller_water_temp_set",
			}
			h.mu.Lock()
			err := h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.Println("err: ", err)
			}
		case v := <-h.changeV:
			h.c.SetV(v)
			reply := model.Msg{
				Type:    "v_set",
				Content: "v_set",
			}
			h.mu.Lock()
			err := h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.Println("err: ", err)
			}
		case <-h.started: // 开始计算
			// 从calculator里面的hub中获取是否有
			h.c.GetCalcHub().StartSignal()
			go h.c.Run()    // 不断计算
			go h.pushData() // 获取推送的计算结果到前端
			reply := model.Msg{
				Type:    "started",
				Content: "Started",
			}
			h.mu.Lock()
			err := h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.Println("err: ", err)
			}
		case <-h.stopped: // 停止计算
			h.c.GetCalcHub().StopSignal()
			reply := model.Msg{
				Type:    "stopped",
				Content: "stopped",
			}
			h.mu.Lock()
			err := h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.Println("err: ", err)
			}
		case <-h.tailStart: // 拉尾坯
			h.c.SetStateTail()
			reply := model.Msg{
				Type:    "tail_start",
				Content: "started to tail",
			}
			h.mu.Lock()
			err := h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.Println("err: ", err)
			}
		case index := <-h.startPushSliceDetail:
			fmt.Println("startPushSliceDetail")
			if h.c.GetCalcHub().PushSliceDetailRunning {
				h.c.GetCalcHub().StopPushSliceDetail()
			}
			h.c.GetCalcHub().PushSliceDetailRunning = true
			go h.c.SliceDetailRun()
			go h.pushSliceDetail(index)
			reply := model.Msg{
				Type:    "start_push_slice_detail_success",
				Content: "start_push_slice_detail_success",
			}
			h.mu.Lock()
			err := h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.Println("err: ", err)
			}
		case <-h.stopPushSliceDetail:
			fmt.Println("stopPushSliceDetail")
			if h.c.GetCalcHub().PushSliceDetailRunning {
				h.c.GetCalcHub().StopPushSliceDetail()
				h.c.GetCalcHub().PushSliceDetailRunning = false
			}
			reply := model.Msg{
				Type:    "stop_push_slice_detail_success",
				Content: "stop_push_slice_detail_success",
			}
			h.mu.Lock()
			err := h.conn.WriteJSON(&reply)
			h.mu.Unlock()
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
					return
				}
				fmt.Println("获取到env参数：", env)
				h.envSet <- env
			case "change_initial_temp":
				temp, err := strconv.ParseFloat(msg.Content, 10)
				if err != nil {
					log.Println("err: ", err)
					return
				}
				fmt.Println("获取到初始温度参数：", temp)
				h.changeInitialTemp <- float32(temp)
			case "change_narrow_surface":
				var narrowSurface model.NarrowSurface
				err := json.Unmarshal([]byte(msg.Content), &narrowSurface)
				if err != nil {
					log.Println("err: ", err)
					return
				}
				fmt.Println("获取到窄面温度参数：", narrowSurface)
				h.changeNarrowSurface <- narrowSurface
			case "change_wide_surface":
				var wideSurface model.WideSurface
				err := json.Unmarshal([]byte(msg.Content), &wideSurface)
				if err != nil {
					log.Println("err: ", err)
					return
				}
				fmt.Println("获取到宽面温度参数：", wideSurface)
				h.changeWideSurface <- wideSurface
			case "change_spray_temp":
				temp, err := strconv.ParseFloat(msg.Content, 10)
				if err != nil {
					log.Println("err: ", err)
					return
				}
				fmt.Println("获取到二冷区喷淋温度：", temp)
				h.changeSprayTemp <- float32(temp)
			case "change_roller_water_temp":
				temp, err := strconv.ParseFloat(msg.Content, 10)
				if err != nil {
					log.Println("err: ", err)
					return
				}
				fmt.Println("获取到二冷区棍子温度：", temp)
				h.changeRollerWaterTemp <- float32(temp)
			case "change_v":
				v, err := strconv.ParseFloat(msg.Content, 10)
				if err != nil {
					log.Println("err: ", err)
					return
				}
				fmt.Println("获取到拉速参数：", v)
				h.changeV <- float32(v)
			case "start":
				h.started <- struct{}{}
			case "stop":
				h.stopped <- struct{}{}
			case "tail":
				h.tailStart <- struct{}{}
			case "start_push_slice_detail":
				index, err := strconv.ParseInt(msg.Content, 10, 64)
				if err != nil {
					log.Println("err: ", err)
					return
				}
				if index < 0 || int(index) >= h.c.GetFieldSize() {
					return
				}
				fmt.Println("获取到切片下标参数：", index)
				h.startPushSliceDetail <- int(index)
			case "stop_push_slice_detail":
				h.stopPushSliceDetail <- struct{}{}
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
LOOP:
	for {
		select {
		case <-h.c.GetCalcHub().Stop:
			break LOOP
		case <-h.c.GetCalcHub().PeriodCalcResult:
			temperatureData := h.c.BuildData()
			data, err := json.Marshal(temperatureData)
			if err != nil {
				log.Println("err: ", err)
				return
			}
			reply.Content = string(data)
			h.mu.Lock()
			err = h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.Println("err: ", err)
			}
		}
	}
}

func (h *Hub) pushSliceDetail(index int) {
	reply := model.Msg{
		Type: "slice_detail",
	}
LOOP:
	for {
		select {
		case <-h.c.GetCalcHub().StopPushSliceDataSignalForPush:
			fmt.Println("stop push slice detail data")
			h.c.GetCalcHub().StopSuccessForPush <- struct{}{}
			break LOOP
		case <-h.c.GetCalcHub().PeriodPushSliceData:
			h.pushSliceData(reply, index)
		}
	}

}

func (h *Hub) pushSliceData(reply model.Msg, index int) {
	sliceData := h.c.BuildSliceData(index)
	data, err := json.Marshal(sliceData)
	if err != nil {
		log.Println("err: ", err)
		return
	}
	reply.Content = string(data)
	h.mu.Lock()
	err = h.conn.WriteJSON(&reply)
	h.mu.Unlock()
	if err != nil {
		log.Println("err: ", err)
	}
}
