package server

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"lz/calculator"
	"lz/model"
	"os"
	"strconv"
	"sync"
	"time"
)

func initLog() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	// Only log the warning severity or above.
	log.SetLevel(log.DebugLevel)
}

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

	generate chan struct{}

	mu sync.Mutex
}

func NewHub() *Hub {
	initLog()
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

		generate: make(chan struct{}, 10),
	}
}

func (h *Hub) handleResponse() {
	defer func() {
		log.Info("停止handleResponse")
	}()
	for {
		select {
		case env := <-h.envSet: // 设置计算环境
			if h.c == nil {
				//c := calculator.NewCalculator(0)
				//c := calculator.NewCalculatorWithListDeque(0)
				h.c = calculator.NewCalculatorWithArrDeque()
			}
			h.c.GetCastingMachine().SetCoolerConfig(env)           // 设置冷却参数
			h.c.GetCastingMachine().SetV(env.DragSpeed)            // 设置拉速
			h.c.InitSteel(env.SteelValue, h.c.GetCastingMachine()) // 设置钢种物性参数
			reply := model.Msg{
				Type:    "env_set",
				Content: "env is set",
			}
			h.mu.Lock()
			err := h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.WithField("err", err).Error("回复消息失败")
			}
		case temp := <-h.changeInitialTemp:
			h.c.GetCastingMachine().SetStartTemperature(temp)
			reply := model.Msg{
				Type:    "initial_temp_set",
				Content: "initial_temp_set",
			}
			h.mu.Lock()
			err := h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.WithField("err", err).Error("回复消息失败")
			}
		case narrowSurface := <-h.changeNarrowSurface:
			h.c.GetCastingMachine().SetNarrowSurfaceIn(narrowSurface.In)
			h.c.GetCastingMachine().SetNarrowSurfaceOut(narrowSurface.Out)
			reply := model.Msg{
				Type:    "narrow_surface_temp_set",
				Content: "narrow_surface_temp_set",
			}
			h.mu.Lock()
			err := h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.WithField("err", err).Error("回复消息失败")
			}
		case wideSurface := <-h.changeWideSurface:
			h.c.GetCastingMachine().SetWideSurfaceIn(wideSurface.In)
			h.c.GetCastingMachine().SetWideSurfaceOut(wideSurface.Out)
			reply := model.Msg{
				Type:    "wide_surface_temp_set",
				Content: "wide_surface_temp_set",
			}
			h.mu.Lock()
			err := h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.WithField("err", err).Error("回复消息失败")
			}
		case temp := <-h.changeSprayTemp:
			h.c.GetCastingMachine().SetSprayTemperature(temp)
			reply := model.Msg{
				Type:    "spray_water_temp_set",
				Content: "spray_water_temp_set",
			}
			h.mu.Lock()
			err := h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.WithField("err", err).Error("回复消息失败")
			}
		case temp := <-h.changeRollerWaterTemp:
			h.c.GetCastingMachine().SetRollerWaterTemperature(temp)
			reply := model.Msg{
				Type:    "roller_water_temp_set",
				Content: "roller_water_temp_set",
			}
			h.mu.Lock()
			err := h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.WithField("err", err).Error("回复消息失败")
			}
		case v := <-h.changeV:
			h.c.GetCastingMachine().SetV(v)
			reply := model.Msg{
				Type:    "v_set",
				Content: "v_set",
			}
			h.mu.Lock()
			err := h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.WithField("err", err).Error("回复消息失败")
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
				log.WithField("err", err).Error("回复消息失败")
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
				log.WithField("err", err).Error("回复消息失败")
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
				log.WithField("err", err).Error("回复消息失败")
			}
		case index := <-h.startPushSliceDetail:
			fmt.Println("startPushSliceDetail")
			if h.c.GetCalcHub().PushSliceDetailRunning {
				h.c.GetCalcHub().StopPushSliceDetail()
			}
			h.c.GetCalcHub().PushSliceDetailRunning = true
			go h.c.GetCalcHub().SliceDetailRun()
			go h.pushSliceDetail(index)
			reply := model.Msg{
				Type:    "start_push_slice_detail_success",
				Content: "start_push_slice_detail_success",
			}
			h.mu.Lock()
			err := h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.WithField("err", err).Error("回复消息失败")
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
				log.WithField("err", err).Error("回复消息失败")
			}
		case <- h.generate:
			reply := model.Msg{
				Type: "data_generate",
			}
			h.c = calculator.NewCalculatorForGenerate()
			log.Info("初始化计算器")
			temperatureData := h.c.GenerateResult()
			log.Info("生成数据")
			data, err := json.Marshal(temperatureData)
			if err != nil {
				log.WithField("err", err).Error("温度场推送数据json解析失败")
				return
			}
			reply.Content = string(data)
			h.mu.Lock()
			err = h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.WithField("err", err).Error("发送温度场推送消息失败")
			}
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (h *Hub) handleRequest() {
	// 可以在此对请求进行预处理
	defer func() {
		fmt.Println("停止handleRequest")
	}()
	for {
		select {
		case msg := <-h.msg:
			switch msg.Type {
			case "env":
				var env model.Env
				err := json.Unmarshal([]byte(msg.Content), &env)
				if err != nil {
					log.Println("err", err)
					return
				}
				log.WithField("env", env).Info("获取到计算环境参数")
				h.envSet <- env
			case "change_initial_temp":
				temp, err := strconv.ParseFloat(msg.Content, 10)
				if err != nil {
					log.Println("err", err)
					return
				}
				log.WithField("temp", temp).Info("获取到初始温度参数")
				h.changeInitialTemp <- float32(temp)
			case "change_narrow_surface":
				var narrowSurface model.NarrowSurface
				err := json.Unmarshal([]byte(msg.Content), &narrowSurface)
				if err != nil {
					log.Println("err", err)
					return
				}
				log.WithField("narrowSurface", narrowSurface).Info("获取到窄面温度参数")
				h.changeNarrowSurface <- narrowSurface
			case "change_wide_surface":
				var wideSurface model.WideSurface
				err := json.Unmarshal([]byte(msg.Content), &wideSurface)
				if err != nil {
					log.Println("err", err)
					return
				}
				log.WithField("wideSurface", wideSurface).Info("获取到宽面温度参数")
				h.changeWideSurface <- wideSurface
			case "change_spray_temp":
				temp, err := strconv.ParseFloat(msg.Content, 10)
				if err != nil {
					log.Println("err", err)
					return
				}
				log.WithField("spray_temp", temp).Info("获取到二冷区喷淋温度")
				h.changeSprayTemp <- float32(temp)
			case "change_roller_water_temp":
				temp, err := strconv.ParseFloat(msg.Content, 10)
				if err != nil {
					log.Println("err", err)
					return
				}
				log.WithField("roller_water_temp", temp).Info("获取到二冷区棍子温度")
				h.changeRollerWaterTemp <- float32(temp)
			case "change_v":
				v, err := strconv.ParseFloat(msg.Content, 10)
				if err != nil {
					log.Println("err", err)
					return
				}
				log.WithField("v", v).Info("获取到拉速参数")
				h.changeV <- float32(v)
			case "start":
				log.Info("开始计算三维温度场")
				h.started <- struct{}{}
			case "stop":
				log.Info("停止计算三维温度场")
				h.stopped <- struct{}{}
			case "tail":
				h.tailStart <- struct{}{}
			case "start_push_slice_detail":
				log.Info("开始计算切片详情")
				index, err := strconv.ParseInt(msg.Content, 10, 64)
				if err != nil {
					log.WithField("err", err).Error("切片下标不是整数")
					return
				}
				if index < 0 || int(index) >= h.c.GetFieldSize() {
					log.Warn("切片下标越界")
					break
				}
				log.WithField("index", index).Info("获取到切片下标参数")
				h.startPushSliceDetail <- int(index)
				log.Info("开始计算切片详情信号发送完毕")
			case "stop_push_slice_detail":
				log.Info("获取到停止推送切片数据的信号")
				h.stopPushSliceDetail <- struct{}{}
			case "generate":
				log.Info("获取到生成数据的信号")
				h.generate <- struct{}{}
			default:
				log.Warn("no such type")
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
				log.WithField("err", err).Error("温度场推送数据json解析失败")
				return
			}
			reply.Content = string(data)
			h.mu.Lock()
			err = h.conn.WriteJSON(&reply)
			h.mu.Unlock()
			if err != nil {
				log.WithField("err", err).Error("发送温度场推送消息失败")
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
			log.Info("停止推送切片详情")
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
		log.WithField("err", err).Error("温度场横切面推送数据json解析失败")
		return
	}
	reply.Content = string(data)
	h.mu.Lock()
	err = h.conn.WriteJSON(&reply)
	h.mu.Unlock()
	if err != nil {
		log.WithField("err", err).Error("发送温度场横切面推送消息失败")
	}
}
