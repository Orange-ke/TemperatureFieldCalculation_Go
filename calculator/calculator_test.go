package calculator

import (
	"fmt"
	"lz/model"
	"runtime"
	"testing"
	"time"
)

func TestCalculator1(t *testing.T) {
	runtime.GOMAXPROCS(12)
	calculator := NewCalculatorWithArrDeque(newExecutorBaseOnBlock(0))
	calculator.castingMachine = NewCastingMachine()
	calculator.GetCastingMachine().SetCoolerConfig(model.Env{
		StartTemperature: 1600.0,
		Md: model.Md{
			NarrowSurfaceIn:  20.0,
			NarrowSurfaceOut: 38.0,
			WideSurfaceIn:    20.0,
			WideSurfaceOut:   38.0,
		},
	})
	calculator.steel1 = NewSteel(1, calculator.castingMachine)
	fmt.Println(calculator.castingMachine.CoolerConfig.StartTemperature)
	calculator.runningState = stateRunning
	calculator.Calculate()
}

func TestCalculator2(t *testing.T) {
	runtime.GOMAXPROCS(12)
	ZLength = 31860
	Length = 1260.0 / 2
	Width = 230.0 / 2
	calculator := NewCalculatorWithArrDeque(nil)
	calculator.castingMachine = NewCastingMachine()
	calculator.GetCastingMachine().SetCoolerConfig(model.Env{
		StartTemperature: 1600.0,
		Md: model.Md{
			NarrowSurfaceIn:     30.0,
			NarrowSurfaceOut:    38.0,
			NarrowSurfaceVolume: 300,
			WideSurfaceIn:       30.0,
			WideSurfaceOut:      38.0,
			WideSurfaceVolume:   3000,
		},
	})
	calculator.castingMachine.SetV(1.5)
	calculator.castingMachine.SetFromJson(model.Coordinate{
		MdLength: 950,
	})
	calculator.steel1 = NewSteel(1, calculator.castingMachine)
	fmt.Println(calculator.castingMachine.CoolerConfig.StartTemperature)
	calculator.runningState = stateRunning
	calculator.Calculate()
}

func TestCalculatorWithArrDeque_calculate(t *testing.T) {
	calculator := NewCalculatorForGenerate()
	calculator.castingMachine = NewCastingMachine()
	calculator.GetCastingMachine().SetCoolerConfig(model.Env{
		StartTemperature: 1600.0,
		Md: model.Md{
			NarrowSurfaceIn:  20.0,
			NarrowSurfaceOut: 38.0,
			WideSurfaceIn:    20.0,
			WideSurfaceOut:   38.0,
		},
	})
	calculator.steel1 = NewSteel(1, calculator.castingMachine)
	fmt.Println(calculator.castingMachine.CoolerConfig.StartTemperature)
	for i := 0; i < 4000; i++ {
		calculator.Field.AddFirst(calculator.castingMachine.CoolerConfig.StartTemperature)
	}
	calculator.runningState = stateRunning
	start := time.Now()
	e := newExecutorBaseOnBlock(0)
	deltaT, _ := calculator.calculateTimeStep()
	e.f[0](task{start: 0, end: calculator.Field.Size(), deltaT: deltaT}, calculator)
	if calculator.alternating {
		calculator.Field = calculator.thermalField1
	} else {
		calculator.Field = calculator.thermalField
	}
	calculator.alternating = !calculator.alternating
	fmt.Println(time.Since(start))
}

func TestCalculateTimeStep(t *testing.T) {
	calculator := NewCalculatorWithArrDeque(nil)
	calculator.calculateTimeStep()
}

//func TestCalculator_Test(t *testing.T) {
//	calculator := NewCalculatorWithArrDeque(0)
//	calculator.Calculate()
//}
