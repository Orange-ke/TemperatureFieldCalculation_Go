package calculator

import (
	"fmt"
	"lz/model"
	"time"
)

// 获取等效步长
func getEx(x int) int {
	if x == 0 || x == Length/XStep-1 {
		return 2 * XStep
	}
	return XStep
}

func getEy(y int) int {
	if y == 0 || y == Width/YStep-1 {
		return 2 * YStep
	}
	return YStep
}

// 计算实际传热系数
func getLambda(index1, index2, x1, y1, x2, y2 int) float32 {
	var K = float32(0.9) // 修正系数K
	// 等效空间步长
	var ex1 = getEx(x1)
	var ex2 = getEx(x2)
	var ey1 = getEy(y1)
	var ey2 = getEy(y2)
	if x1 != x2 {
		return K * Lambda[index1] * Lambda[index2] * float32(ex1+ex2) /
			(Lambda[index1]*float32(ex2) + Lambda[index2]*float32(ex1))
	}
	if y1 != y2 {
		return K * Lambda[index1] * Lambda[index2] * float32(ey1+ey2) /
			(Lambda[index1]*float32(ey2) + Lambda[index2]*float32(ey1))
	}
	return 1.0 // input error
}

// 计算时间步长 ------------------------------------------------------------------------------------------------------------------
// 计算时间步长 case1 -> 左下角
func getDeltaTCase1(x, y int, slice *model.ItemType) float32 {
	var t = slice[y][x]
	var index = int(t)/5 - 1
	var index1, index2 int
	index1 = int(slice[y][x+1])/5 - 1
	index2 = int(slice[y+1][x])/5 - 1
	denominator := 2*getLambda(index, index1, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index2, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1)))
	return (Density[index] * Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case2 -> 下面边
func getDeltaTCase2(x, y int, slice *model.ItemType) float32 {
	var t = slice[y][x]
	var index = int(t)/5 - 1
	var index1, index2, index3 int
	index1 = int(slice[y][x-1])/5 - 1
	index2 = int(slice[y][x+1])/5 - 1
	index3 = int(slice[y+1][x])/5 - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index3, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1)))
	return (Density[index] * Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case3 -> 右下角
func getDeltaTCase3(x, y int, slice *model.ItemType) float32 {
	var t = slice[y][x]
	var index = int(t)/5 - 1
	var index1, index2 int
	index1 = int(slice[y][x-1])/5 - 1
	index2 = int(slice[y+1][x])/5 - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1))) +
		HEff[index]/(XStep)
	return (Density[index] * Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case4 -> 右面边
func getDeltaTCase4(x, y int, slice *model.ItemType) float32 {
	var t = slice[y][x]
	var index = int(t)/5 - 1
	var index1, index2, index3 int
	index1 = int(slice[y][x-1])/5 - 1
	index2 = int(slice[y+1][x])/5 - 1
	index3 = int(slice[y-1][x])/5 - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1))) +
		2*getLambda(index, index3, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1))) +
		HEff[index]/(XStep)
	return (Density[index] * Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case5 -> 右上角
func getDeltaTCase5(x, y int, slice *model.ItemType) float32 {
	var t = slice[y][x]
	var index = int(t)/5 - 1
	var index1, index2 int
	index1 = int(slice[y][x-1])/5 - 1
	index2 = int(slice[y-1][x])/5 - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1))) +
		HEff[index]/(XStep) +
		HEff[index]/(YStep)
	return (Density[index] * Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case6 -> 上面边
func getDeltaTCase6(x, y int, slice *model.ItemType) float32 {
	var t = slice[y][x]
	var index = int(t)/5 - 1
	var index1, index2, index3 int
	index1 = int(slice[y][x-1])/5 - 1
	index2 = int(slice[y][x+1])/5 - 1
	index3 = int(slice[y-1][x])/5 - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index3, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1))) +
		HEff[index]/(YStep)
	return (Density[index] * Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case7 -> 左上角
func getDeltaTCase7(x, y int, slice *model.ItemType) float32 {
	var t = slice[y][x]
	var index = int(t)/5 - 1
	var index1, index2 int
	index1 = int(slice[y][x+1])/5 - 1
	index2 = int(slice[y-1][x])/5 - 1
	denominator := 2*getLambda(index, index1, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index2, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1))) +
		HEff[index]/(YStep)
	return (Density[index] * Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case8 -> 左面边
func getDeltaTCase8(x, y int, slice *model.ItemType) float32 {
	var t = slice[y][x]
	var index = int(t)/5 - 1
	var index1, index2, index3 int
	index1 = int(slice[y][x+1])/5 - 1
	index2 = int(slice[y+1][x])/5 - 1
	index3 = int(slice[y-1][x])/5 - 1
	denominator := 2*getLambda(index, index1, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index2, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1))) +
		2*getLambda(index, index3, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1)))
	return (Density[index] * Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case9 -> 内部点
func getDeltaTCase9(x, y int, slice *model.ItemType) float32 {
	var t = slice[y][x]
	var index = int(t)/5 - 1
	var index1, index2, index3, index4 int
	index1 = int(slice[y][x-1])/5 - 1
	index2 = int(slice[y][x+1])/5 - 1
	index3 = int(slice[y+1][x])/5 - 1
	index4 = int(slice[y-1][x])/5 - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index3, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1))) +
		2*getLambda(index, index4, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1)))
	return (Density[index] * Enthalpy[index]) / (t * denominator)
}

// 计算一个切片的时间步长
func calculateTimeStepOfOneSlice(slice *model.ItemType) float32 {
	// 计算时间步长 - start
	var deltaTArr = [9]float32{}
	deltaTArr[0] = getDeltaTCase1(0, 0, slice)
	deltaTArr[1] = getDeltaTCase2(Length/XStep-2, 0, slice)
	deltaTArr[2] = getDeltaTCase3(Length/XStep-1, 0, slice)
	deltaTArr[3] = getDeltaTCase4(Length/XStep-1, Width/YStep-2, slice)
	deltaTArr[4] = getDeltaTCase5(Length/XStep-1, Width/YStep-1, slice)
	deltaTArr[5] = getDeltaTCase6(Length/XStep-2, Width/YStep-1, slice)
	deltaTArr[6] = getDeltaTCase7(0, Width/YStep-1, slice)
	deltaTArr[7] = getDeltaTCase8(0, Width/YStep-2, slice)
	deltaTArr[8] = getDeltaTCase9(Length/XStep-2, Width/YStep-2, slice)
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
	startTime := time.Now()
	// up
	for y := Width/YStep - 1; y >= 0; y -= stepY {
		for x := Length/XStep - 1; x >= 0; x -= stepX {
			temperatureData.Up.Up[Width/YStep/2+y/stepY][Length/XStep/2+x/stepX] = ThermalField.Field[ThermalField.Start][y][x]
			temperatureData.Up.Up[Width/YStep/2-1-y/stepY][Length/XStep/2-1-x/stepX] = ThermalField.Field[ThermalField.Start][y][x]
			temperatureData.Up.Up[Width/YStep/2+y/stepY][Length/XStep/2-1-x/stepX] = ThermalField.Field[ThermalField.Start][y][x]
			temperatureData.Up.Up[Width/YStep/2-1-y/stepY][Length/XStep/2+x/stepX] = ThermalField.Field[ThermalField.Start][y][x]
		}
	}
	zStart := ThermalField.Start
	zEnd := upLength
	if ThermalField.End < zEnd {
		zEnd = ThermalField.End
	}
	for y := Width/YStep - 1; y >= 0; y -= stepY {
		for x := zEnd - 1; x >=  zStart; x -= stepZ {
			temperatureData.Up.Left[Width/YStep/2+y/stepY][x/stepZ] = ThermalField.Field[x][y][Length/XStep-1]
			temperatureData.Up.Left[Width/YStep/2-1-y/stepY][x/stepZ] = ThermalField.Field[x][y][Length/XStep-1]
		}
	}
	for y := Width/YStep - 1; y >= 0; y -= stepY {
		for x := zEnd - 1; x >=  zStart; x -= stepZ {
			temperatureData.Up.Right[Width/YStep/2+y/stepY][x/stepZ] = ThermalField.Field[x][y][Length/XStep-1]
			temperatureData.Up.Right[Width/YStep/2-1-y/stepY][x/stepZ] = ThermalField.Field[x][y][Length/XStep-1]
		}
	}
	for y := Length/XStep - 1; y >= 0; y -= stepX {
		for x := zEnd - 1; x >=  zStart; x -= stepZ {
			temperatureData.Up.Front[Length/XStep/2+y/stepX][x/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
			temperatureData.Up.Front[Length/XStep/2-y/stepX-1][x/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
		}
	}
	for y := Length/XStep - 1; y >= 0; y -= stepX {
		for x := zEnd - 1; x >=  zStart; x -= stepZ {
			temperatureData.Up.Back[Length/XStep/2+y/stepX][x/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
			temperatureData.Up.Back[Length/XStep/2-y/stepX-1][x/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
		}
	}

	zStart = upLength
	zEnd = upLength + arcLength
	if ThermalField.End < zEnd {
		zEnd = ThermalField.End
	}
	for y := Width/YStep - 1; y >= 0; y -= stepY {
		for x := zEnd - 1; x >=  zStart; x -= stepZ {
			temperatureData.Arc.Left[Width/YStep/2+y/stepY][(x-zStart)/stepZ] = ThermalField.Field[x][y][Length/XStep-1]
			temperatureData.Arc.Left[Width/YStep/2-1-y/stepY][(x-zStart)/stepZ] = ThermalField.Field[x][y][Length/XStep-1]
		}
	}
	for y := Width/YStep - 1; y >= 0; y -= stepY {
		for x := zEnd - 1; x >=  zStart; x -= stepZ {
			temperatureData.Arc.Right[Width/YStep/2+y/stepY][(x-zStart)/stepZ] = ThermalField.Field[x][y][Length/XStep-1]
			temperatureData.Arc.Right[Width/YStep/2-1-y/stepY][(x-zStart)/stepZ] = ThermalField.Field[x][y][Length/XStep-1]
		}
	}
	for y := Length/XStep - 1; y >= 0; y -= stepX {
		for x := zEnd - 1; x >=  zStart; x -= stepZ {
			temperatureData.Arc.Front[Length/XStep/2+y/stepX][(x-zStart)/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
			temperatureData.Arc.Front[Length/XStep/2-y/stepX-1][(x-zStart)/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
		}
	}
	for y := Length/XStep - 1; y >= 0; y -= stepX {
		for x := zEnd - 1; x >=  zStart; x -= stepZ {
			temperatureData.Arc.Back[Length/XStep/2+y/stepX][(x-zStart)/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
			temperatureData.Arc.Back[Length/XStep/2-y/stepX-1][(x-zStart)/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
		}
	}

	zStart = upLength + arcLength
	zEnd = upLength + arcLength + downLength
	if ThermalField.End < zEnd {
		zEnd = ThermalField.End
	}
	for y := Width/YStep - 1; y >= 0; y -= stepY {
		for x := Length/XStep - 1; x >= 0; x -= stepX {
			temperatureData.Down.Down[Width/YStep/2+y/stepY][Length/XStep/2+x/stepX] = ThermalField.Field[ThermalField.End-1][y][x]
			temperatureData.Down.Down[Width/YStep/2-1-y/stepY][Length/XStep/2-1-x/stepX] = ThermalField.Field[ThermalField.End-1][y][x]
			temperatureData.Down.Down[Width/YStep/2+y/stepY][Length/XStep/2-1-x/stepX] = ThermalField.Field[ThermalField.End-1][y][x]
			temperatureData.Down.Down[Width/YStep/2-1-y/stepY][Length/XStep/2+x/stepX] = ThermalField.Field[ThermalField.End-1][y][x]
		}
	}
	for y := Width/YStep - 1; y >= 0; y -= stepY {
		for x := zEnd - 1; x >=  zStart; x -= stepZ {
			temperatureData.Down.Left[Width/YStep/2+y/stepY][(x-zStart)/stepZ] = ThermalField.Field[x][y][Length/XStep-1]
			temperatureData.Down.Left[Width/YStep/2-y/stepY-1][(x-zStart)/stepZ] = ThermalField.Field[x][y][Length/XStep-1]
		}
	}
	for y := Width/YStep - 1; y >= 0; y -= stepY {
		for x := zEnd - 1; x >=  zStart; x -= stepZ {
			temperatureData.Down.Right[Width/YStep/2+y/stepY][(x-zStart)/stepZ] = ThermalField.Field[x][y][Length/XStep-1]
			temperatureData.Down.Right[Width/YStep/2-y/stepY-1][(x-zStart)/stepZ] = ThermalField.Field[x][y][Length/XStep-1]
		}
	}
	for y := Length/XStep - 1; y >= 0; y -= stepX {
		for x := zEnd - 1; x >=  zStart; x -= stepZ {
			temperatureData.Down.Front[Length/XStep/2+y/stepX][(x-zStart)/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
			temperatureData.Down.Front[Length/XStep/2-y/stepX-1][(x-zStart)/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
		}
	}
	for y := Length/XStep - 1; y >= 0; y -= stepX {
		for x := zEnd - 1; x >=  zStart; x -= stepZ {
			temperatureData.Down.Back[Length/XStep/2+y/stepX][(x-zStart)/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
			temperatureData.Down.Back[Length/XStep/2-y/stepX-1][(x-zStart)/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
		}
	}

	//fmt.Printf("up up: 长%d 宽%d")
	fmt.Println("build data cost: ", time.Since(startTime))
	// temperatureData
}
