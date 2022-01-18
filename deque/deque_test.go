package deque

import (
	"fmt"
	"lz/calculator"
	"lz/model"
	"testing"
	"time"
)

func TestArrDeque_Traverse(t *testing.T) {
	deque := NewArrDeque(4000)
	for i := 0; i < 4000; i++ {
		deque.AddLast([calculator.Width / calculator.YStep][calculator.Length / calculator.XStep]float32{})
	}
	start := time.Now()
	for c := 0; c < 100; c++ {
		deque.Traverse(func(z int, item *model.ItemType) {
			//fmt.Printf("%p \n", item)
			for i := 0; i < len(item); i++ {
				for j := 0; j < len(item[0]); j++ {
					item[i][j] += 1
				}
			}
		})
	}

	fmt.Println(time.Since(start))
}

func TestListDeque_Traverse(t *testing.T) {
	deque := NewListDeque()
	for i := 0; i < 4000; i++ {
		deque.AddLast([calculator.Width / calculator.YStep][calculator.Length / calculator.XStep]float32{})
	}
	start := time.Now()
	for c := 0; c < 100; c++ {
		deque.Traverse(func(z int,item *model.ItemType) {
			//fmt.Printf("%p \n", item)
			for i := 0; i < len(item); i++ {
				for j := 0; j < len(item[0]); j++ {
					item[i][j]++
				}
			}
		})
	}
	fmt.Println(time.Since(start))
}

func TestArr_Traverse(t *testing.T) {
	const zLength = 4000
	arr := make([][42][270]float32, zLength)
	start := time.Now()
	for c := 0; c < 100; c++ {
		for z := 0; z <= zLength-20; z += 20 {
			for i := 0; i < 42; i++ {
				for j := 0; j < 270; j++ {
					arr[z][i][j]++
					arr[z+1][i][j]++
					arr[z+2][i][j]++
					arr[z+3][i][j]++
					arr[z+4][i][j]++
					arr[z+5][i][j]++
					arr[z+6][i][j]++
					arr[z+7][i][j]++
					arr[z+8][i][j]++
					arr[z+9][i][j]++
					arr[z+10][i][j]++
					arr[z+11][i][j]++
					arr[z+12][i][j]++
					arr[z+13][i][j]++
					arr[z+14][i][j]++
					arr[z+15][i][j]++
					arr[z+16][i][j]++
					arr[z+17][i][j]++
					arr[z+18][i][j]++
					arr[z+19][i][j]++
				}
			}
		}
	}
	fmt.Println(time.Since(start))
}

//func min(x, y int) int {
//	if x < y {
//		return x
//	}
//	return y
//}
