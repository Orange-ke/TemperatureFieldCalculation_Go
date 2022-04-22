package calculator

import (
	"testing"
)

type EncodedPushData2 struct {
	Top    Encoder2
	Arc    Encoder2
	Bottom Encoder2
}

type Encoder2 struct {
	Start int
	Data  []int
}

func TestPrint(t *testing.T) {
	e := newEncoder()
	middle := e.GeneratePushData2()
	final := EncodedPushData2{
		Top: Encoder2{
			Data: make([]int, 0),
		},
		Arc: Encoder2{
			Data: make([]int, 0),
		},
		Bottom: Encoder2{
			Data: make([]int, 0),
		},
	}
	final.Top.Start = middle.Top[0]
	for i := 1; i < len(middle.Top); i++ {
		final.Top.Data = append(final.Top.Data, middle.Top[i]-final.Top.Start)
	}
	final.Arc.Start = middle.Arc[0]
	for i := 1; i < len(middle.Arc); i++ {
		final.Arc.Data = append(final.Arc.Data, middle.Arc[i]-final.Arc.Start)
	}
	final.Bottom.Start = middle.Bottom[0]
	for i := 1; i < len(middle.Bottom); i++ {
		final.Bottom.Data = append(final.Bottom.Data, middle.Bottom[i]-final.Bottom.Start)
	}

	m := make(map[int]int, 10)
	for i := 0; i < len(final.Top.Data); i++ {
		m[final.Top.Data[i]]++
	}
	for i := 0; i < len(final.Arc.Data); i++ {
		m[final.Arc.Data[i]]++
	}
	for i := 0; i < len(final.Bottom.Data); i++ {
		m[final.Bottom.Data[i]]++
	}

	var leaves []*Node
	for k, v := range m {
		leaves = append(leaves, &Node{
			Value: ValueType(k),
			Count: v,
		})
	}

	root := Build(leaves)
	Print(root)
}

func TestPrint1(t *testing.T) {
	e := newEncoder()
	middle := e.GeneratePushData1()

	m := make(map[int8]int, 10)
	for i := 0; i < len(middle.Top.Data); i++ {
		m[middle.Top.Data[i]]++
	}
	for i := 0; i < len(middle.Arc.Data); i++ {
		m[middle.Arc.Data[i]]++
	}
	for i := 0; i < len(middle.Bottom.Data); i++ {
		m[middle.Bottom.Data[i]]++
	}

	var leaves []*Node
	for k, v := range m {
		leaves = append(leaves, &Node{
			Value: ValueType(k),
			Count: v,
		})
	}

	root := Build(leaves)
	Print(root)
}
