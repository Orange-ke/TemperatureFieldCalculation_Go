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
		StartTemperature: 1530.0,
		Md: model.Md{
			NarrowSurfaceIn:     30.0,
			NarrowSurfaceOut:    38.0,
			NarrowSurfaceVolume: 540,
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
	//calculator.Calculate(
	calculator.TestCalculateQ()

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

func TestCalculatorExecutor(t *testing.T) {
	ZLength = 31860
	Length = 1260.0 / 2
	Width = 230.0 / 2
	calculator := NewCalculatorWithArrDeque(nil)
	for z := 0; z < 1; z++ {
		calculator.thermalField.AddFirst(0)
		calculator.thermalField1.AddFirst(0)
	}
	count := 0
	calculator.Field.TraverseSpirally(0, 1, func(z int, item *model.ItemType) {
		// 跳过为空的切片， 即值为-1
		if item[0][0] == -1 {
			return
		}
		left, right, top, bottom := 0, Length/XStep-1, 0, Width/YStep-1 // 每个切片迭代时需要重置
		// 计算最外层， 逆时针
		{
			// 1. 三个顶点，左下方顶点仅当其外一层温度不是初始温度时才开始计算
			//item[0][Length/XStep-1] = 1
			//item[Width/YStep-1][Length/XStep-1] = 1
			//item[Width/YStep-1][0] = 1
			count += 3
			for row := top + 1; row < bottom; row++ {
				// [row][right]
				//item[row][Length/XStep-1] = 1
				count++
			}
			for column := right - 1; column > left; column-- {
				// [bottom][column]
				//item[Width/YStep-1][column] = 1
				count++
			}
			right--
			bottom--
		}

		{
			// 逆时针螺旋遍历
			for left <= right && top <= bottom {
				//if item[0][right] != item[0][right+1] ||
				//	item[0][right] != item[0][right-1] ||
				//	item[0][right] != item[1][right] {
				//	item[0][right] = 1
				//	count++
				//}
				//item[0][right] = 1
				count++
				for row := top + 1; row <= bottom; row++ {
					// [row][right]
					//if item[row][right] != item[row][right+1] ||
					//	item[row][right] != item[row][right-1] ||
					//	item[row][right] != item[row+1][right] ||
					//	item[row][right] != item[row-1][right] {
					//	item[row][right] = 1
					//	count++
					//}
					//item[row][right] = 1
					count++
				}
				if left < right && top < bottom {
					for column := right - 1; column > left; column-- {
						// [bottom][column]
						//if item[bottom][column] != item[bottom][column+1] ||
						//	item[bottom][column] != item[bottom][column-1] ||
						//	item[bottom][column] != item[bottom+1][column] ||
						//	item[bottom][column] != item[bottom-1][column] {
						//	item[bottom][column] = 1
						//	count++
						//}
						//item[bottom][column] = 1
						count++
						//if item[bottom][0] == item[bottom+1][0] ||
						//	item[bottom][0] == item[bottom-1][0] ||
						//	item[bottom][0] == item[bottom][1] {
						//	item[bottom][0] = 1
						//	count++
						//}
					}
					//item[bottom][0] = 1
					count++
				}
				if top == bottom {
					//if item[0][0] != item[0][1] || item[0][0] != item[1][0] {
					//	item[0][0] = 1
					//	count++
					//}
					//item[0][0] = 1
					count++
					for column := right - 1; column > left; column-- {
						//if item[0][column] != item[0][column+1] ||
						//	item[0][column] != item[0][column-1] ||
						//	item[0][column] != item[1][column] {
						//	item[0][column] = 1
						//	count++
						//}
						//item[0][column] = 1
						count++
					}
				}
				right--
				bottom--
			}
		}
	})
	if !calculator.Field.IsEmpty() {
		for i := Width/YStep - 1; i >= 0; i-- {
			for j := 0; j <= Length/XStep-1; j++ {
				fmt.Printf("%.2f ", calculator.Field.Get(calculator.Field.Size()-1, i, j))
			}
			fmt.Println()
		}
	}
	fmt.Println("计算的点数: ", count, "实际需要遍历的点数: ", Width/YStep*Length/XStep)
}
