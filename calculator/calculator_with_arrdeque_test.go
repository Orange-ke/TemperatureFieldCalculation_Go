package calculator

import (
	"fmt"
	"lz/model"
	"testing"
)

func TestNewCalculatorWithArrDeque(t *testing.T) {
	c := NewCalculatorWithArrDeque(nil)

	for i := 0; i < 4000; i++ {
		c.thermalField.AddFirst(c.castingMachine.CoolerConfig.StartTemperature - 200.0 + float32(i) * 0.025)
		c.thermalField1.AddFirst(c.castingMachine.CoolerConfig.StartTemperature - 200.0 + float32(i) * 0.025)
	}

	fmt.Println(c.isFull, c.thermalField.IsFull(), c.thermalField1.IsFull(), c.alternating)

	for i := 0; i < 100; i++ {
		deltaT, _ := c.calculateTimeStep()
		c.Field.Traverse(func(z int, item *model.ItemType) {
			parameter := c.getParameter(z)
			c.calculatePointRT(deltaT, z, item, parameter)
		}, 0, 0)

		if c.alternating {
			c.Field = c.thermalField1
		} else {
			c.Field = c.thermalField
		}

		c.alternating = !c.alternating // 仅在这里修改
		fmt.Println(c.Field.Get(c.Field.Size() - 1, 41, 269))
	}
	fmt.Println("-----------------------")
	for i := 0; i < 10; i++ {
		deltaT, _ := c.calculateTimeStep()
		c.Field.Traverse(func(z int, item *model.ItemType) {
			parameter := c.getParameter(z)
			c.calculatePointRT(deltaT, z, item, parameter)
		}, 0, 0)

		for k := 0; k < 100; k++ {
			c.thermalField.RemoveLast()
			c.thermalField1.RemoveLast()
			c.thermalField.AddFirst(c.castingMachine.CoolerConfig.StartTemperature)
			c.thermalField1.AddFirst(c.castingMachine.CoolerConfig.StartTemperature)
		}

		if c.alternating {
			c.Field = c.thermalField1
		} else {
			c.Field = c.thermalField
		}

		c.alternating = !c.alternating // 仅在这里修改
		fmt.Println(c.Field.Get(c.Field.Size() - 1, 41, 269))
	}
}

func TestCalculatorWithArrDeque_Calculate(t *testing.T) {
	c := NewCalculatorWithArrDeque(nil)
	c.Calculate()
}
