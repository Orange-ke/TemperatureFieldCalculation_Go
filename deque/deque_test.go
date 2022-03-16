package deque

import (
	"fmt"
	"lz/model"
	"testing"
	"time"
)

func TestArrDeque_Traverse(t *testing.T) {
	deque := NewArrDeque(4000)
	for i := 0; i < 4000; i++ {
		deque.AddFirst(1550.0)
	}
	start := time.Now()
	for c := 0; c < 100; c++ {
		deque.Traverse(func(z int, item *model.ItemType) {
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
	deque := NewListDeque(4000)
	for i := 0; i < 4000; i++ {
		deque.AddLast(1550.0)
	}
	start := time.Now()
	for c := 0; c < 100; c++ {
		deque.Traverse(func(z int,item *model.ItemType) {
			for i := 0; i < len(item); i++ {
				for j := 0; j < len(item[0]); j++ {
					item[i][j]++
				}
			}
		})
	}
	fmt.Println(time.Since(start))
}

func BenchmarkArrDeque_AddFirst(b *testing.B) {
	deque := NewArrDeque(4000)
	for i := 0; i < b.N; i++ {
		deque.AddFirst(1000)
		deque.RemoveFirst()
	}
}

func BenchmarkArrDeque_RemoveLast(b *testing.B) {
	deque := NewArrDeque(4000)
	for i := 0; i < b.N; i++ {
		deque.AddLast(1000)
		deque.RemoveLast()
	}
}

func BenchmarkListDeque_AddFirst(b *testing.B) {
	deque := NewListDeque(4000)
	for i := 0; i < b.N; i++ {
		deque.AddFirst(1000)
		deque.RemoveFirst()
	}
}

func BenchmarkListDeque_AddLast(b *testing.B) {
	deque := NewListDeque(4000)
	for i := 0; i < b.N; i++ {
		deque.AddLast(1000)
		deque.RemoveLast()
	}
}

// 测试循环展开
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

func TestArrDeque_Funcs(t *testing.T) {
	deque := NewArrDeque(4000)
	for i := 0; i < 4000; i++ {
		deque.AddFirst(1550.0)
	}
	fmt.Println(deque.container.start, deque.container.end, deque.container1.start, deque.container1.end, deque.container.isFull, deque.container1.isFull, deque.isFull, deque.state)
	deque.RemoveLast()
	fmt.Println(deque.container.start, deque.container.end, deque.container1.start, deque.container1.end, deque.container.isFull, deque.container1.isFull, deque.isFull, deque.state)
	deque.AddFirst(1550.0)
	fmt.Println(deque.container.start, deque.container.end, deque.container1.start, deque.container1.end, deque.container.isFull, deque.container1.isFull, deque.isFull, deque.state)
	deque.AddFirst(1550.0)
	fmt.Println(deque.container.start, deque.container.end, deque.container1.start, deque.container1.end, deque.container.isFull, deque.container1.isFull, deque.isFull, deque.state)
	deque.RemoveLast()
	fmt.Println(deque.container.start, deque.container.end, deque.container1.start, deque.container1.end, deque.container.isFull, deque.container1.isFull, deque.isFull, deque.state)
	deque.AddFirst(1550.0)
	fmt.Println(deque.container.start, deque.container.end, deque.container1.start, deque.container1.end, deque.container.isFull, deque.container1.isFull, deque.isFull, deque.state)
	deque.RemoveLast()
	fmt.Println(deque.container.start, deque.container.end, deque.container1.start, deque.container1.end, deque.container.isFull, deque.container1.isFull, deque.isFull, deque.state)
	deque.AddFirst(1550.0)
	fmt.Println(deque.IsFull())
	fmt.Println(deque.container.start, deque.container.end, deque.container1.start, deque.container1.end, deque.container.isFull, deque.container1.isFull, deque.isFull, deque.state)

	deque.Set(deque.Size() - 1, 41, 269, 1490, 0)
	fmt.Println(deque.Get(deque.Size() - 1, 41, 269))
}
