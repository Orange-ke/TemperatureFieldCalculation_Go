package calculator

import (
	"fmt"
	"lz/model"
	"time"
)

// 获取等效步长
func getEx(x int) int {
	if x == 0 || x == model.Length/model.XStep-1 {
		return 2 * model.XStep
	}
	return model.XStep
}

func getEy(y int) int {
	if y == 0 || y == model.Width/model.YStep-1 {
		return 2 * model.YStep
	}
	return model.YStep
}

// 计算实际传热系数
func getLambda(index1, index2, x1, y1, x2, y2 int, parameter *parameter) float32 {
	var K = float32(0.9) // 修正系数K
	// 等效空间步长
	var ex1 = getEx(x1)
	var ex2 = getEx(x2)
	var ey1 = getEy(y1)
	var ey2 = getEy(y2)
	if x1 != x2 {
		return K * parameter.Lambda[index1] * parameter.Lambda[index2] * float32(ex1+ex2) /
			(parameter.Lambda[index1]*float32(ex2) + parameter.Lambda[index2]*float32(ex1))
	}
	if y1 != y2 {
		return K * parameter.Lambda[index1] * parameter.Lambda[index2] * float32(ey1+ey2) /
			(parameter.Lambda[index1]*float32(ey2) + parameter.Lambda[index2]*float32(ey1))
	}
	return 1.0 // input error
}

// 计算时间步长 ------------------------------------------------------------------------------------------------------------------
// 计算时间步长 case1 -> 左下角
func getDeltaTCase1(x, y int, slice *model.ItemType, parameter *parameter) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2 int
	index1 = int(slice[y][x+1]) - 1
	index2 = int(slice[y+1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x+1, y, parameter)/float32(model.XStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index2, x, y, x, y+1, parameter)/float32(model.YStep*(getEy(y)+getEy(y+1)))
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case2 -> 下面边
func getDeltaTCase2(x, y int, slice *model.ItemType, parameter *parameter) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2, index3 int
	index1 = int(slice[y][x-1]) - 1
	index2 = int(slice[y][x+1]) - 1
	index3 = int(slice[y+1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y, parameter)/float32(model.XStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x+1, y, parameter)/float32(model.XStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index3, x, y, x, y+1, parameter)/float32(model.YStep*(getEy(y)+getEy(y+1)))
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case3 -> 右下角
func getDeltaTCase3(x, y int, slice *model.ItemType, parameter *parameter) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2 int
	index1 = int(slice[y][x-1]) - 1
	index2 = int(slice[y+1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y, parameter)/float32(model.XStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x, y+1, parameter)/float32(model.YStep*(getEy(y)+getEy(y+1))) +
		parameter.GetHeff(t, parameter)/(model.XStep)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case4 -> 右面边
func getDeltaTCase4(x, y int, slice *model.ItemType, parameter *parameter) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2, index3 int
	index1 = int(slice[y][x-1]) - 1
	index2 = int(slice[y+1][x]) - 1
	index3 = int(slice[y-1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y, parameter)/float32(model.XStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x, y+1, parameter)/float32(model.YStep*(getEy(y)+getEy(y+1))) +
		2*getLambda(index, index3, x, y, x, y-1, parameter)/float32(model.YStep*(getEy(y)+getEy(y-1))) +
		parameter.GetHeff(t, parameter)/(model.XStep)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case5 -> 右上角
func getDeltaTCase5(x, y int, slice *model.ItemType, parameter *parameter) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2 int
	index1 = int(slice[y][x-1]) - 1
	index2 = int(slice[y-1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y, parameter)/float32(model.XStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x, y-1, parameter)/float32(model.YStep*(getEy(y)+getEy(y-1))) +
		parameter.GetHeff(t, parameter)/(model.XStep) +
		parameter.GetHeff(t, parameter)/(model.YStep)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case6 -> 上面边
func getDeltaTCase6(x, y int, slice *model.ItemType, parameter *parameter) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2, index3 int
	index1 = int(slice[y][x-1]) - 1
	index2 = int(slice[y][x+1]) - 1
	index3 = int(slice[y-1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y, parameter)/float32(model.XStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x+1, y, parameter)/float32(model.XStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index3, x, y, x, y-1, parameter)/float32(model.YStep*(getEy(y)+getEy(y-1))) +
		parameter.GetHeff(t, parameter)/(model.YStep)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case7 -> 左上角
func getDeltaTCase7(x, y int, slice *model.ItemType, parameter *parameter) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2 int
	index1 = int(slice[y][x+1]) - 1
	index2 = int(slice[y-1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x+1, y, parameter)/float32(model.XStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index2, x, y, x, y-1, parameter)/float32(model.YStep*(getEy(y)+getEy(y-1))) +
		parameter.GetHeff(t, parameter)/(model.YStep)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case8 -> 左面边
func getDeltaTCase8(x, y int, slice *model.ItemType, parameter *parameter) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2, index3 int
	index1 = int(slice[y][x+1]) - 1
	index2 = int(slice[y+1][x]) - 1
	index3 = int(slice[y-1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x+1, y, parameter)/float32(model.XStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index2, x, y, x, y+1, parameter)/float32(model.YStep*(getEy(y)+getEy(y+1))) +
		2*getLambda(index, index3, x, y, x, y-1, parameter)/float32(model.YStep*(getEy(y)+getEy(y-1)))
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case9 -> 内部点
func getDeltaTCase9(x, y int, slice *model.ItemType, parameter *parameter) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2, index3, index4 int
	index1 = int(slice[y][x-1]) - 1
	index2 = int(slice[y][x+1]) - 1
	index3 = int(slice[y+1][x]) - 1
	index4 = int(slice[y-1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y, parameter)/float32(model.XStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x+1, y, parameter)/float32(model.XStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index3, x, y, x, y+1, parameter)/float32(model.YStep*(getEy(y)+getEy(y+1))) +
		2*getLambda(index, index4, x, y, x, y-1, parameter)/float32(model.YStep*(getEy(y)+getEy(y-1)))
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算一个切片的时间步长
func calculateTimeStepOfOneSlice(slice *model.ItemType, parameter *parameter) float32 {
	// 计算时间步长 - start
	var deltaTArr = [9]float32{}
	deltaTArr[0] = getDeltaTCase1(0, 0, slice, parameter)
	deltaTArr[1] = getDeltaTCase2(model.Length/model.XStep-2, 0, slice, parameter)
	deltaTArr[2] = getDeltaTCase3(model.Length/model.XStep-1, 0, slice, parameter)
	deltaTArr[3] = getDeltaTCase4(model.Length/model.XStep-1, model.Width/model.YStep-2, slice, parameter)
	deltaTArr[4] = getDeltaTCase5(model.Length/model.XStep-1, model.Width/model.YStep-1, slice, parameter)
	deltaTArr[5] = getDeltaTCase6(model.Length/model.XStep-2, model.Width/model.YStep-1, slice, parameter)
	deltaTArr[6] = getDeltaTCase7(0, model.Width/model.YStep-1, slice, parameter)
	deltaTArr[7] = getDeltaTCase8(0, model.Width/model.YStep-2, slice, parameter)
	deltaTArr[8] = getDeltaTCase9(model.Length/model.XStep-2, model.Width/model.YStep-2, slice, parameter)
	var min = float32(1000.0) // 模拟一个很大的数
	for _, i := range deltaTArr {
		if min > i {
			min = i
		}
	}
	return min
	// 计算时间步长 - end
}

// 计算时间步长 ------------------------------------------------------------------------------------------------------------------

// 构建push data
func buildDataHelper(ThermalField *ThermalFieldStruct, temperatureData *TemperatureData) {
	// 跳过为空的切片
	for ThermalField.Field[ThermalField.Start][0][0] == -1 {
		ThermalField.Start++
		if ThermalField.Start == model.ZLength / model.ZStep {
			return
		}
	}
	startTime := time.Now()
	// up
	for y := model.Width/model.YStep - 1; y >= 0; y -= stepY {
		for x := model.Length/model.XStep - 1; x >= 0; x -= stepX {
			temperatureData.Up.Up[model.Width/model.YStep/2+y/stepY][model.Length/model.XStep/2+x/stepX] = ThermalField.Field[ThermalField.Start][y][x]
			temperatureData.Up.Up[model.Width/model.YStep/2-1-y/stepY][model.Length/model.XStep/2-1-x/stepX] = ThermalField.Field[ThermalField.Start][y][x]
			temperatureData.Up.Up[model.Width/model.YStep/2+y/stepY][model.Length/model.XStep/2-1-x/stepX] = ThermalField.Field[ThermalField.Start][y][x]
			temperatureData.Up.Up[model.Width/model.YStep/2-1-y/stepY][model.Length/model.XStep/2+x/stepX] = ThermalField.Field[ThermalField.Start][y][x]
		}
	}
	start := 0
	zStart := ThermalField.Start
	zEnd := upLength
	if ThermalField.End < zEnd {
		zEnd = ThermalField.End
	}
	for y := model.Width/model.YStep - 1; y >= 0; y -= stepY {
		for x := zEnd - 1; x >= zStart; x -= stepZ {
			temperatureData.Up.Left[model.Width/model.YStep/2+y/stepY][x/stepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
			temperatureData.Up.Left[model.Width/model.YStep/2-1-y/stepY][x/stepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
		}
	}
	for y := model.Width/model.YStep - 1; y >= 0; y -= stepY {
		for x := zEnd - 1; x >= zStart; x -= stepZ {
			temperatureData.Up.Right[model.Width/model.YStep/2+y/stepY][x/stepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
			temperatureData.Up.Right[model.Width/model.YStep/2-1-y/stepY][x/stepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
		}
	}
	for y := model.Length/model.XStep - 1; y >= 0; y -= stepX {
		for x := zEnd - 1; x >= zStart; x -= stepZ {
			temperatureData.Up.Front[model.Length/model.XStep/2+y/stepX][x/stepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
			temperatureData.Up.Front[model.Length/model.XStep/2-y/stepX-1][x/stepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
		}
	}
	for y := model.Length/model.XStep - 1; y >= 0; y -= stepX {
		for x := zEnd - 1; x >= zStart; x -= stepZ {
			temperatureData.Up.Back[model.Length/model.XStep/2+y/stepX][x/stepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
			temperatureData.Up.Back[model.Length/model.XStep/2-y/stepX-1][x/stepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
		}
	}

	start = upLength
	zStart = max(upLength, ThermalField.Start)
	zEnd = upLength + arcLength
	if ThermalField.End < zEnd {
		zEnd = ThermalField.End
	}
	for y := model.Width/model.YStep - 1; y >= 0; y -= stepY {
		for x := zEnd - 1; x >= zStart; x -= stepZ {
			temperatureData.Arc.Left[model.Width/model.YStep/2+y/stepY][(x-start)/stepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
			temperatureData.Arc.Left[model.Width/model.YStep/2-1-y/stepY][(x-start)/stepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
		}
	}
	for y := model.Width/model.YStep - 1; y >= 0; y -= stepY {
		for x := zEnd - 1; x >= zStart; x -= stepZ {
			temperatureData.Arc.Right[model.Width/model.YStep/2+y/stepY][(x-start)/stepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
			temperatureData.Arc.Right[model.Width/model.YStep/2-1-y/stepY][(x-start)/stepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
		}
	}
	for y := model.Length/model.XStep - 1; y >= 0; y -= stepX {
		for x := zEnd - 1; x >= zStart; x -= stepZ {
			temperatureData.Arc.Front[model.Length/model.XStep/2+y/stepX][(x-start)/stepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
			temperatureData.Arc.Front[model.Length/model.XStep/2-y/stepX-1][(x-start)/stepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
		}
	}
	for y := model.Length/model.XStep - 1; y >= 0; y -= stepX {
		for x := zEnd - 1; x >= zStart; x -= stepZ {
			temperatureData.Arc.Back[model.Length/model.XStep/2+y/stepX][(x-start)/stepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
			temperatureData.Arc.Back[model.Length/model.XStep/2-y/stepX-1][(x-start)/stepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
		}
	}

	start = upLength + arcLength
	zStart = max(upLength + arcLength, ThermalField.Start)
	zEnd = upLength + arcLength + downLength
	if ThermalField.End < zEnd {
		zEnd = ThermalField.End
	}
	for y := model.Width/model.YStep - 1; y >= 0; y -= stepY {
		for x := model.Length/model.XStep - 1; x >= 0; x -= stepX {
			temperatureData.Down.Down[model.Width/model.YStep/2+y/stepY][model.Length/model.XStep/2+x/stepX] = ThermalField.Field[ThermalField.End-1][y][x]
			temperatureData.Down.Down[model.Width/model.YStep/2-1-y/stepY][model.Length/model.XStep/2-1-x/stepX] = ThermalField.Field[ThermalField.End-1][y][x]
			temperatureData.Down.Down[model.Width/model.YStep/2+y/stepY][model.Length/model.XStep/2-1-x/stepX] = ThermalField.Field[ThermalField.End-1][y][x]
			temperatureData.Down.Down[model.Width/model.YStep/2-1-y/stepY][model.Length/model.XStep/2+x/stepX] = ThermalField.Field[ThermalField.End-1][y][x]
		}
	}
	for y := model.Width/model.YStep - 1; y >= 0; y -= stepY {
		for x := zEnd - 1; x >= zStart; x -= stepZ {
			temperatureData.Down.Left[model.Width/model.YStep/2+y/stepY][(x-start)/stepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
			temperatureData.Down.Left[model.Width/model.YStep/2-y/stepY-1][(x-start)/stepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
		}
	}
	for y := model.Width/model.YStep - 1; y >= 0; y -= stepY {
		for x := zEnd - 1; x >= zStart; x -= stepZ {
			temperatureData.Down.Right[model.Width/model.YStep/2+y/stepY][(x-start)/stepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
			temperatureData.Down.Right[model.Width/model.YStep/2-y/stepY-1][(x-start)/stepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
		}
	}
	for y := model.Length/model.XStep - 1; y >= 0; y -= stepX {
		for x := zEnd - 1; x >= zStart; x -= stepZ {
			temperatureData.Down.Front[model.Length/model.XStep/2+y/stepX][(x-start)/stepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
			temperatureData.Down.Front[model.Length/model.XStep/2-y/stepX-1][(x-start)/stepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
		}
	}
	for y := model.Length/model.XStep - 1; y >= 0; y -= stepX {
		for x := zEnd - 1; x >= zStart; x -= stepZ {
			temperatureData.Down.Back[model.Length/model.XStep/2+y/stepX][(x-start)/stepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
			temperatureData.Down.Back[model.Length/model.XStep/2-y/stepX-1][(x-start)/stepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
		}
	}

	//fmt.Printf("up up: 长%d 宽%d")
	fmt.Println("build data cost: ", time.Since(startTime))
	// temperatureData
}

// 获取热流密度和综合换热系数
// s <= r
func (c *calculatorWithArrDeque) getHeffLessThanR(T float32, parameter *parameter) float32 {
	return parameter.Q[int(T)] / (T - c.coolerConfig.NarrowSurfaceIn)
}

func (c *calculatorWithArrDeque) getQLessThanR(T float32, parameter *parameter) float32 {
	return parameter.Q[int(T)]
}

// s > r
func (c *calculatorWithArrDeque) getHeffGreaterThanR(T float32, parameter *parameter) float32 {
	return parameter.HEff[int(T)]
}

func (c *calculatorWithArrDeque) getQGreaterThanR(T float32, parameter *parameter) float32 {
	return parameter.HEff[int(T)] * (T - c.coolerConfig.SprayTemperature)
}

func max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
