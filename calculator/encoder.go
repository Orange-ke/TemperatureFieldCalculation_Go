package calculator

import (
	"fmt"
	"lz/model"
	"math"
	"time"
)

// 第一步先将数据进行处理
type encoder struct {
}

type EncodedPushData struct {
	Top    Encoder
	Arc    Encoder
	Bottom Encoder
}

type Encoder struct {
	Start int
	Data  []int8
}

type MiddleState struct {
	Top    []int
	Arc    []int
	Bottom []int
}

func newEncoder() *encoder {
	return &encoder{}
}

func (e *encoder) GeneratePushData1() EncodedPushData {
	c := NewCalculatorForGenerate()
	res := c.GenerateResultForEncoder()
	//printTop(res)
	//printArc(res)
	printBottom(res)
	start := time.Now()
	final := EncodedPushData{}
	final.Top = generateMiddleData(res.Top)
	final.Arc = generateMiddleData(res.Arc)
	final.Bottom = generateMiddleData(res.Bottom)
	//fmt.Println(len(final.Top.Data), len(final.Arc.Data), len(final.Bottom.Data))
	//fmt.Println(final.Top)
	//fmt.Println(final.Bottom)
	fmt.Println(time.Since(start), "1321312")
	return final
}

func (e *encoder) GeneratePushData2() *MiddleState {
	c := NewCalculatorForGenerate()
	for i := 0; i < 4000; i++ {
		c.Field.AddFirst(1600.0)
	}
	return c.GenerateResultForEncoder()
}

func generateMiddleData(data []int) Encoder {
	max := math.MinInt32
	res := make([]int8, 0)
	first := data[0]
	pre := first
	start := first
	index := 0
	length := 0
	gap := 0
	for index < len(data) {
		if start == data[index] {
			for index < len(data) && start == data[index] {
				index++
				length++
				res = append(res, int8(gap))
				if index < len(data) && gap > max {
					max = gap
				}
			}
		} else {
			gap = Abs(data[index] - pre)
			pre = data[index]
			start = data[index]
			length = 0
		}
	}
	return Encoder{
		Start: first,
		Data: res,
	}
}

func printTop(res *MiddleState) {
	fmt.Println("-----------------------------top--------------------------------")
	fmt.Println(len(res.Top))
	lenTop := model.Length/model.XStep + model.Width/model.YStep - 1
	index := 0
	fmt.Println(len(res.Top))
	for index < len(res.Top) {
		for i := 0; i < lenTop; i++ {
			fmt.Printf("%4d ", res.Top[i+index])
		}
		index += lenTop
		fmt.Println()
		lenTop -= 2
	}
	fmt.Println(index)
}

func printArc(res *MiddleState) {
	fmt.Println(len(res.Arc))
	index := 0
	fmt.Println("-----------------------------arc--------------------------------")
	lenArc := model.Length/model.XStep + model.Width/model.YStep - 1
	for index < len(res.Arc) {
		for i := 0; i < lenArc; i++ {
			fmt.Printf("%4d ", res.Arc[i+index])
		}
		fmt.Println()
		index += lenArc
	}
	fmt.Println(index)
}

func printBottom(res *MiddleState) {
	fmt.Println(len(res.Bottom))
	index := 0
	lenBottom := model.Length/model.XStep + model.Width/model.YStep - 1
	fmt.Println("-----------------------------bottom--------------------------------")
	for index < len(res.Bottom) {
		for i := 0; i < lenBottom; i++ {
			fmt.Printf("%4d ", res.Bottom[i+index])
		}
		index += lenBottom
		fmt.Println()
		lenBottom -= 2
	}
	fmt.Println(index)
}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func decode(src Encoder) []int {
	res := make([]int, 0)
	start := src.Start
	for i := 0; i < len(src.Data); i++ {
		res = append(res, start + int(src.Data[i]))
		start = start + int(src.Data[i])
	}
	return res
}