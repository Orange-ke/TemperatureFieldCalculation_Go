package server

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"lz/calculator"
	"lz/model"
	"time"
)

const (
	upLength   = 500
	downLength = 500
	arcLength  = 3000

	stepX = 2
	stepY = 2
	stepZ = 10
)

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	c    calculator.Calculator
	conn *websocket.Conn
	// request
	msg chan model.Msg
	// response
	envSet  chan model.Msg
	started chan model.Msg
	stopped chan model.Msg
}

func NewHub() *Hub {
	return &Hub{
		msg:     make(chan model.Msg, 10),
		envSet:  make(chan model.Msg, 10),
		started: make(chan model.Msg, 10),
		stopped: make(chan model.Msg, 10),
	}
}

func (h *Hub) handleResponse() {
	for {
		select {
		case reply := <-h.envSet:
			err := h.conn.WriteJSON(&reply)
			if err != nil {
				log.Println("err: ", err)
			}
		case reply := <-h.started:
			h.c.CalculateConcurrently()
			temperatureData := h.buildData()
			data, err := json.Marshal(temperatureData)
			if err != nil {
				log.Println("err: ", err)
			}
			reply.Content = string(data)
			err = h.conn.WriteJSON(&reply)
			if err != nil {
				log.Println("err: ", err)
			}
		case reply := <-h.stopped:
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
				reply := model.Msg{
					Type:    "envSet",
					Content: "env is set",
					// 可以加一些需要的参数过去
				}
				h.envSet <- reply
			case "start":
				reply := model.Msg{
					Type: "started",
				}
				h.started <- reply
			case "stop":
				reply := model.Msg{
					Type:    "stopped",
					Content: "stopped",
				}
				h.stopped <- reply
			default:
				log.Println("no such type")
			}
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

type TemperatureData struct {
	Up   UpSides `json:"up"`
	Arc  ArcSides `json:"arc"`
	Down DownSides `json:"down"`
}

type UpSides struct {
	Up    [calculator.Width / calculator.YStep / stepY][calculator.Length / calculator.XStep / stepX]float32 `json:"up"`
	Left  [calculator.Width / calculator.YStep / stepY][upLength / stepZ]float32 `json:"left"`
	Right [calculator.Width / calculator.YStep / stepY][upLength / stepZ]float32 `json:"right"`
	Front [calculator.Length / calculator.XStep / stepX][upLength / stepZ]float32 `json:"front"`
	Back  [calculator.Length / calculator.XStep / stepX][upLength / stepZ]float32 `json:"back"`
}

type ArcSides struct {
	Left  [calculator.Width / calculator.YStep / stepY][arcLength / stepZ]float32 `json:"left"`
	Right [calculator.Width / calculator.YStep / stepY][arcLength / stepZ]float32 `json:"right"`
	Front [calculator.Length / calculator.XStep / stepX][arcLength / stepZ]float32 `json:"front"`
	Back  [calculator.Length / calculator.XStep / stepX][arcLength / stepZ]float32 `json:"back"`
}

type DownSides struct {
	Left  [calculator.Width / calculator.YStep / stepY][upLength / stepZ]float32 `json:"left"`
	Right [calculator.Width / calculator.YStep / stepY][upLength / stepZ]float32 `json:"right"`
	Front [calculator.Length / calculator.XStep  / stepX][upLength / stepZ]float32 `json:"front"`
	Back  [calculator.Length / calculator.XStep  / stepX][upLength / stepZ]float32 `json:"back"`
	Down  [calculator.Width / calculator.YStep / stepY][calculator.Length / calculator.XStep / stepX]float32 `json:"down"`
}

func (h *Hub) buildData() TemperatureData {
	upSides := UpSides{}
	for y := 0; y < calculator.Width/calculator.YStep; y+=stepY {
		for x := 0; x < calculator.Length/calculator.XStep; x+=stepX {
			upSides.Up[y/stepY][x/stepX] = h.c.ThermalField[0][y][x]
		}
	}
	for y := 0; y < calculator.Width/calculator.YStep; y+=stepY {
		for x := 0; x < upLength; x+=stepZ {
			upSides.Left[y/stepY][x/stepZ] = h.c.ThermalField[x][y][0]
		}
	}
	for y := 0; y < calculator.Width/calculator.YStep; y+=stepY {
		for x := 0; x < upLength; x+=stepZ {
			upSides.Right[y/stepY][x/stepZ] = h.c.ThermalField[x][y][calculator.Length/calculator.XStep-1]
		}
	}
	for y := 0; y < calculator.Length/calculator.XStep; y+=stepX {
		for x := 0; x < upLength; x+=stepZ {
			upSides.Front[y/stepX][x/stepZ] = h.c.ThermalField[x][calculator.Width/calculator.YStep-1][y]
		}
	}
	for y := 0; y < calculator.Length/calculator.XStep; y+=stepX {
		for x := 0; x < upLength; x+=stepZ {
			upSides.Back[y/stepX][x/stepZ] = h.c.ThermalField[x][0][y]
		}
	}

	arcSides := ArcSides{}
	zStart := upLength
	zEnd := upLength + arcLength
	for y := 0; y < calculator.Width/calculator.YStep; y+=stepY {
		for x := zStart; x < zEnd; x+=stepZ {
			arcSides.Left[y/stepY][(x - zStart)/stepZ] = h.c.ThermalField[x][y][0]
		}
	}
	for y := 0; y < calculator.Width/calculator.YStep; y+=stepY {
		for x := zStart; x < zEnd; x+=stepZ {
			arcSides.Right[y/stepY][(x - zStart)/stepZ] = h.c.ThermalField[x][y][calculator.Length/calculator.XStep-1]
		}
	}
	for y := 0; y < calculator.Length/calculator.XStep; y+=stepX {
		for x := zStart; x < zEnd; x+=stepZ {
			arcSides.Front[y/stepX][(x - zStart)/stepZ] = h.c.ThermalField[x][calculator.Width/calculator.YStep-1][y]
		}
	}
	for y := 0; y < calculator.Length/calculator.XStep; y+=stepX {
		for x := zStart; x < zEnd; x+=stepZ {
			arcSides.Back[y/stepX][(x - zStart)/stepZ] = h.c.ThermalField[x][0][y]
		}
	}

	downSides := DownSides{}
	zStart = upLength + arcLength
	zEnd = upLength + arcLength + downLength
	for y := 0; y < calculator.Width/calculator.YStep; y+=stepY {
		for x := 0; x < calculator.Length/calculator.XStep; x+=stepX {
			downSides.Down[y/stepY][x/stepX] = h.c.ThermalField[zEnd-1][y][x]
		}
	}
	for y := 0; y < calculator.Width/calculator.YStep; y+=stepY {
		for x := zStart; x < zEnd; x+=stepZ {
			downSides.Left[y/stepY][(x - zStart)/stepZ] = h.c.ThermalField[x][y][0]
		}
	}
	for y := 0; y < calculator.Width/calculator.YStep; y+=stepY {
		for x := zStart; x < zEnd; x+=stepZ {
			downSides.Right[y/stepY][(x - zStart)/stepZ] = h.c.ThermalField[x][y][calculator.Length/calculator.XStep-1]
		}
	}
	for y := 0; y < calculator.Length/calculator.XStep; y+=stepX {
		for x := zStart; x < zEnd; x+=stepZ {
			downSides.Front[y/stepX][(x - zStart)/stepZ] = h.c.ThermalField[x][calculator.Width/calculator.YStep-1][y]
		}
	}
	for y := 0; y < calculator.Length/calculator.XStep; y+=stepX {
		for x := zStart; x < zEnd; x+=stepZ {
			downSides.Back[y/stepX][(x - zStart)/stepZ] = h.c.ThermalField[x][0][y]
		}
	}

	return TemperatureData{
		upSides,
		arcSides,
		downSides,
	}
}
