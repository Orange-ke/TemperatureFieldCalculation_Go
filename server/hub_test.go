package server

import (
	"encoding/json"
	"fmt"
	"lz/calculator"
	"lz/model"
	"testing"
	"time"
)

func TestCalculatorPushTime(t *testing.T) {
	h := NewHub()
	reply := model.Msg{
		Type: "data_push",
	}
	h.c = calculator.NewCalculatorForGenerate()
	temperatureData := h.c.GenerateResult()
	data, _ := json.Marshal(temperatureData)
	total := time.Second * 0
	max := time.Second * 0
	for i := 0; i < 100; i++ {
		start := time.Now()
		reply.Content = string(data)
		_ = h.conn.WriteJSON(&reply)
		total += time.Since(start)
		if max < time.Since(start) {
			max = time.Since(start)
		}
	}
	fmt.Print("切片充满时传输100次需要的平均时间：", (total / 100).Milliseconds(), "ms")
	fmt.Print("切片充满时传输100次其中最长的一次传输时间：", max.Milliseconds(), "ms")
}