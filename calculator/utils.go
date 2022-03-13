package calculator

import (
	"lz/casting_machine"
	"lz/steel_type"
	"time"
)

// 标准单位为m 将mm 转化为m * 1000
var (
	stdXStep = float32(XStep) / 1000
	stdYStep = float32(YStep) / 1000
)

// 获取等效步长
func getEx(x int) float32 {
	if x == 0 || x == Length/XStep-1 {
		return 2 * stdXStep
	}
	return stdXStep
}

func getEy(y int) float32 {
	if y == 0 || y == Width/YStep-1 {
		return 2 * stdYStep
	}
	return stdYStep
}

// 计算实际传热系数
func getLambda(index1, index2, x1, y1, x2, y2 int, parameter *steel_type.Parameter) float32 {
	var K = float32(0.956) // 修正系数K
	// 等效空间步长
	var ex1 = getEx(x1)
	var ex2 = getEx(x2)
	var ey1 = getEy(y1)
	var ey2 = getEy(y2)
	if x1 != x2 {
		//fmt.Println("计算到的lambda值: ", K * parameter.Lambda[index1] * parameter.Lambda[index2] * float32(ex1+ex2) /
		//	(parameter.Lambda[index1]*float32(ex2) + parameter.Lambda[index2]*float32(ex1)), "坐标为：", x1, y1, x2, y2, index1, index2)
		return K * parameter.Lambda[index1] * parameter.Lambda[index2] * (ex1 + ex2) /
			(parameter.Lambda[index1]*(ex2) + parameter.Lambda[index2]*(ex1))
	}
	if y1 != y2 {
		//fmt.Println("计算到的lambda值: ", K * parameter.Lambda[index1] * parameter.Lambda[index2] * float32(ey1+ey2) /
		//	(parameter.Lambda[index1]*float32(ey2) + parameter.Lambda[index2]*float32(ey1)), "坐标为：", x1, y1, x2, y2, index1, index2)
		return K * parameter.Lambda[index1] * parameter.Lambda[index2] * (ey1 + ey2) /
			(parameter.Lambda[index1]*(ey2) + parameter.Lambda[index2]*(ey1))
	}
	return 1.0 // input error
}

// 计算时间步长 ------------------------------------------------------------------------------------------------------------------
// 计算时间步长 case1 -> 左下角
func getDeltaTCase1(x, y int, slice *casting_machine.ItemType, parameter *steel_type.Parameter) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2 int
	index1 = int(slice[y][x+1]) - 1
	index2 = int(slice[y+1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x+1, y, parameter)/(stdXStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index2, x, y, x, y+1, parameter)/(stdYStep*(getEy(y)+getEy(y+1)))
	//fmt.Println("denominator", denominator, parameter.Density[index]*parameter.Enthalpy[index], "t: ", t)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case2 -> 下面边
func getDeltaTCase2(x, y int, slice *casting_machine.ItemType, parameter *steel_type.Parameter) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2, index3 int
	index1 = int(slice[y][x-1]) - 1
	index2 = int(slice[y][x+1]) - 1
	index3 = int(slice[y+1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y, parameter)/(stdXStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x+1, y, parameter)/(stdXStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index3, x, y, x, y+1, parameter)/(stdYStep*(getEy(y)+getEy(y+1)))
	//fmt.Println("denominator", denominator, parameter.Density[index]*parameter.Enthalpy[index], "t: ", t)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case3 -> 右下角
func getDeltaTCase3(x, y int, slice *casting_machine.ItemType, parameter *steel_type.Parameter) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2 int
	index1 = int(slice[y][x-1]) - 1
	index2 = int(slice[y+1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y, parameter)/(stdXStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x, y+1, parameter)/(stdYStep*(getEy(y)+getEy(y+1))) +
		parameter.GetHeff(t)/(stdXStep)
	//fmt.Println("denominator", denominator, parameter.Density[index]*parameter.Enthalpy[index], "t: ", t)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case4 -> 右面边
func getDeltaTCase4(x, y int, slice *casting_machine.ItemType, parameter *steel_type.Parameter) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2, index3 int
	index1 = int(slice[y][x-1]) - 1
	index2 = int(slice[y+1][x]) - 1
	index3 = int(slice[y-1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y, parameter)/(stdXStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x, y+1, parameter)/(stdYStep*(getEy(y)+getEy(y+1))) +
		2*getLambda(index, index3, x, y, x, y-1, parameter)/(stdYStep*(getEy(y)+getEy(y-1))) +
		parameter.GetHeff(t)/(stdXStep)
	//fmt.Println("denominator", denominator, parameter.Density[index]*parameter.Enthalpy[index], "t: ", t)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case5 -> 右上角
func getDeltaTCase5(x, y int, slice *casting_machine.ItemType, parameter *steel_type.Parameter) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2 int
	index1 = int(slice[y][x-1]) - 1
	index2 = int(slice[y-1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y, parameter)/(stdXStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x, y-1, parameter)/(stdYStep*(getEy(y)+getEy(y-1))) +
		parameter.GetHeff(t)/stdXStep +
		parameter.GetHeff(t)/stdYStep
	//fmt.Println(2*getLambda(index, index1, x, y, x-1, y, parameter),
	//	stdXStep,
	//	getEx(x)+getEx(x-1),
	//	2*getLambda(index, index2, x, y, x, y-1, parameter),
	//	stdYStep,
	//	getEy(y)+getEy(y-1),
	//	parameter.GetHeff(t),
	//	parameter.GetHeff(t),
	//	t,
	//)
	//fmt.Println("denominator", denominator, parameter.Density[index]*parameter.Enthalpy[index], "t: ", t)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case6 -> 上面边
func getDeltaTCase6(x, y int, slice *casting_machine.ItemType, parameter *steel_type.Parameter) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2, index3 int
	index1 = int(slice[y][x-1]) - 1
	index2 = int(slice[y][x+1]) - 1
	index3 = int(slice[y-1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y, parameter)/(stdXStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x+1, y, parameter)/(stdXStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index3, x, y, x, y-1, parameter)/(stdYStep*(getEy(y)+getEy(y-1))) +
		parameter.GetHeff(t)/(stdYStep)
	//fmt.Println("denominator", denominator, parameter.Density[index]*parameter.Enthalpy[index], "t: ", t)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case7 -> 左上角
func getDeltaTCase7(x, y int, slice *casting_machine.ItemType, parameter *steel_type.Parameter) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2 int
	index1 = int(slice[y][x+1]) - 1
	index2 = int(slice[y-1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x+1, y, parameter)/(stdXStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index2, x, y, x, y-1, parameter)/(stdYStep*(getEy(y)+getEy(y-1))) +
		parameter.GetHeff(t)/(stdYStep)
	//fmt.Println("denominator", denominator, parameter.Density[index]*parameter.Enthalpy[index], "t: ", t)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case8 -> 左面边
func getDeltaTCase8(x, y int, slice *casting_machine.ItemType, parameter *steel_type.Parameter) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2, index3 int
	index1 = int(slice[y][x+1]) - 1
	index2 = int(slice[y+1][x]) - 1
	index3 = int(slice[y-1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x+1, y, parameter)/(stdXStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index2, x, y, x, y+1, parameter)/(stdYStep*(getEy(y)+getEy(y+1))) +
		2*getLambda(index, index3, x, y, x, y-1, parameter)/(stdYStep*(getEy(y)+getEy(y-1)))
	//fmt.Println("denominator", denominator, parameter.Density[index]*parameter.Enthalpy[index], "t: ", t)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case9 -> 内部点
func getDeltaTCase9(x, y int, slice *casting_machine.ItemType, parameter *steel_type.Parameter) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2, index3, index4 int
	index1 = int(slice[y][x-1]) - 1
	index2 = int(slice[y][x+1]) - 1
	index3 = int(slice[y+1][x]) - 1
	index4 = int(slice[y-1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y, parameter)/(stdXStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x+1, y, parameter)/(stdXStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index3, x, y, x, y+1, parameter)/(stdYStep*(getEy(y)+getEy(y+1))) +
		2*getLambda(index, index4, x, y, x, y-1, parameter)/(stdYStep*(getEy(y)+getEy(y-1)))
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

const bigNum = float32(1000.0)

// 计算一个切片的时间步长
func calculateTimeStepOfOneSlice(slice *casting_machine.ItemType, parameter *steel_type.Parameter) float32 {
	// 计算时间步长 - start
	var deltaTArr = [9]float32{}
	deltaTArr[0] = getDeltaTCase1(0, 0, slice, parameter)
	deltaTArr[1] = getDeltaTCase2(Length/XStep-2, 0, slice, parameter)
	deltaTArr[2] = getDeltaTCase3(Length/XStep-1, 0, slice, parameter)
	deltaTArr[3] = getDeltaTCase4(Length/XStep-1, Width/YStep-2, slice, parameter)
	deltaTArr[4] = getDeltaTCase5(Length/XStep-1, Width/YStep-1, slice, parameter)
	deltaTArr[5] = getDeltaTCase6(Length/XStep-2, Width/YStep-1, slice, parameter)
	deltaTArr[6] = getDeltaTCase7(0, Width/YStep-1, slice, parameter)
	deltaTArr[7] = getDeltaTCase8(0, Width/YStep-2, slice, parameter)
	deltaTArr[8] = getDeltaTCase9(Length/XStep-2, Width/YStep-2, slice, parameter)
	//fmt.Println("时间步长结果：", deltaTArr)
	var min = bigNum // 模拟一个很大的数
	for _, i := range deltaTArr {
		if min > i {
			min = i
		}
	}
	return min
	// 计算时间步长 - end
}

// 计算时间步长 ------------------------------------------------------------------------------------------------------------------

func max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func abs(x time.Duration) time.Duration {
	if x < 0 {
		return -x
	}
	return x
}
