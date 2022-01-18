package calculator

import (
	"fmt"
	"lz/model"
	"testing"
)

func TestNewCalculatorWithArrDeque(t *testing.T) {
	c := NewCalculatorWithArrDeque(0)
	c.thermalField.AddLast(c.initialTemperature)
	c.thermalField.Traverse(func(z int, item *model.ItemType) {
		for i := 0; i < len(item); i++ {
			for j := 0; j < len(item[0]); j++ {
				item[i][j]++
			}
		}
	})

	fmt.Println(c.thermalField.Size(), c.thermalField.Get(0, 0 ,0))

	c.thermalField.RemoveLast()
	fmt.Println(c.thermalField.Size())
}

func TestCalculatorWithArrDeque_Calculate(t *testing.T) {
	c := NewCalculatorWithArrDeque(0)
	c.Calculate()
}