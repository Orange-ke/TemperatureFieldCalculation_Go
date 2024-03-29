package calculator

import (
	"encoding/json"
	"io/ioutil"
	"lz/model"
	"math"
)

// 标准单位为m 将mm 转化为m * 1000
var (
	stdXStep = float32(XStep) / 1000
	stdYStep = float32(YStep) / 1000
)

// 获取等效步长
func getEx(x int) float32 {
	//if x == 0 || x == Length/XStep-1 {
	//	return 2 * stdXStep
	//}
	return stdXStep
}

func getEy(y int) float32 {
	//if y == 0 || y == Width/YStep-1 {
	//	return 2 * stdYStep
	//}
	return stdYStep
}

// 计算实际传热系数
func getLambda(index1, index2, x1, y1, x2, y2 int, parameter *Parameter, zone int, electromagneticStirringFactor float32) float32 {
	electromagneticStirringFactor = 1.0
	var K float32 // 修正系数K
	if zone == Zone0 { // 结晶器
		K = parameter.K[index1 + 1]
	} else {
		K = 1.0 * electromagneticStirringFactor
	}
	//fmt.Println("修正系数K: ", K)
	// 等效空间步长
	var ex1 = getEx(x1)
	var ex2 = getEx(x2)
	var ey1 = getEy(y1)
	var ey2 = getEy(y2)
	if x1 != x2 {
		//fmt.Println("计算到的lambda值: ", K*parameter.Lambda[index1]*parameter.Lambda[index2]*(ex1+ex2)/
		//	(parameter.Lambda[index1]*(ex2)+parameter.Lambda[index2]*(ex1)), "坐标为：", x1, y1, x2, y2, index1, index2)
		return K * parameter.Lambda[index1] * parameter.Lambda[index2] * (ex1 + ex2) /
			(parameter.Lambda[index1]*(ex2) + parameter.Lambda[index2]*(ex1))
	}
	if y1 != y2 {
		//fmt.Println("计算到的lambda值: ", K*parameter.Lambda[index1]*parameter.Lambda[index2]*(ey1+ey2)/
		//	(parameter.Lambda[index1]*(ey2)+parameter.Lambda[index2]*(ey1)), "坐标为：", x1, y1, x2, y2, index1, index2)
		return K * parameter.Lambda[index1] * parameter.Lambda[index2] * (ey1 + ey2) /
			(parameter.Lambda[index1]*(ey2) + parameter.Lambda[index2]*(ey1))
	}
	return 1.0 // input error
}

// 计算时间步长 ------------------------------------------------------------------------------------------------------------------
// 计算时间步长 case1 -> 左下角
func getDeltaTCase1(x, y int, slice *model.ItemType, parameter *Parameter, zone int, electromagneticStirringFactor float32) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2 int
	index1 = int(slice[y][x+1]) - 1
	index2 = int(slice[y+1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x+1, y, parameter, zone, electromagneticStirringFactor)/(stdXStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index2, x, y, x, y+1, parameter, zone, electromagneticStirringFactor)/(stdYStep*(getEy(y)+getEy(y+1)))
	//fmt.Println(getLambda(index, index1, x, y, x+1, y, parameter), stdXStep*(getEx(x)+getEx(x+1)), getLambda(index, index2, x, y, x, y+1, parameter), stdYStep*(getEy(y)+getEy(y+1)))
	//fmt.Println("denominator", denominator, parameter.Density[index]*parameter.Enthalpy[index], "t: ", t)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case2 -> 下面边
func getDeltaTCase2(x, y int, slice *model.ItemType, parameter *Parameter, zone int, electromagneticStirringFactor float32) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2, index3 int
	index1 = int(slice[y][x-1]) - 1
	index2 = int(slice[y][x+1]) - 1
	index3 = int(slice[y+1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y, parameter, zone, electromagneticStirringFactor )/(stdXStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x+1, y, parameter, zone, electromagneticStirringFactor)/(stdXStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index3, x, y, x, y+1, parameter, zone, electromagneticStirringFactor)/(stdYStep*(getEy(y)+getEy(y+1)))
	//fmt.Println("denominator", denominator, parameter.Density[index]*parameter.Enthalpy[index], "t: ", t)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case3 -> 右下角
func getDeltaTCase3(x, y, z int, slice *model.ItemType, parameter *Parameter, zone int, electromagneticStirringFactor float32) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2 int
	index1 = int(slice[y][x-1]) - 1
	index2 = int(slice[y+1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y, parameter, zone, electromagneticStirringFactor)/(stdXStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x, y+1, parameter, zone, electromagneticStirringFactor)/(stdYStep*(getEy(y)+getEy(y+1))) +
		parameter.GetHeff(x, y, z)/(stdXStep)
	//fmt.Println("denominator", denominator, parameter.Density[index]*parameter.Enthalpy[index], "t: ", t)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case4 -> 右面边
func getDeltaTCase4(x, y, z int, slice *model.ItemType, parameter *Parameter, zone int, electromagneticStirringFactor float32) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2, index3 int
	index1 = int(slice[y][x-1]) - 1
	index2 = int(slice[y+1][x]) - 1
	index3 = int(slice[y-1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y, parameter, zone, electromagneticStirringFactor)/(stdXStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x, y+1, parameter, zone, electromagneticStirringFactor)/(stdYStep*(getEy(y)+getEy(y+1))) +
		2*getLambda(index, index3, x, y, x, y-1, parameter, zone, electromagneticStirringFactor)/(stdYStep*(getEy(y)+getEy(y-1))) +
		parameter.GetHeff(x, y, z)/(stdXStep)
	//fmt.Println("denominator", denominator, parameter.Density[index]*parameter.Enthalpy[index], "t: ", t)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case5 -> 右上角
func getDeltaTCase5(x, y, z int, slice *model.ItemType, parameter *Parameter, zone int, electromagneticStirringFactor float32) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2 int
	index1 = int(slice[y][x-1]) - 1
	index2 = int(slice[y-1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y, parameter, zone, electromagneticStirringFactor)/(stdXStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x, y-1, parameter, zone, electromagneticStirringFactor)/(stdYStep*(getEy(y)+getEy(y-1))) +
		parameter.GetHeff(x, y, z)/(stdXStep) +
		parameter.GetHeff(x, y, z)/(stdYStep)
	//fmt.Println("denominator", denominator, parameter.Density[index]*parameter.Enthalpy[index], "t: ", t)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case6 -> 上面边
func getDeltaTCase6(x, y, z int, slice *model.ItemType, parameter *Parameter, zone int, electromagneticStirringFactor float32) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2, index3 int
	index1 = int(slice[y][x-1]) - 1
	index2 = int(slice[y][x+1]) - 1
	index3 = int(slice[y-1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y, parameter, zone, electromagneticStirringFactor)/(stdXStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x+1, y, parameter, zone, electromagneticStirringFactor)/(stdXStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index3, x, y, x, y-1, parameter, zone, electromagneticStirringFactor)/(stdYStep*(getEy(y)+getEy(y-1))) +
		parameter.GetHeff(x, y, z)/(stdYStep)
	//fmt.Println("denominator", denominator, parameter.Density[index]*parameter.Enthalpy[index], "t: ", t)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case7 -> 左上角
func getDeltaTCase7(x, y, z int, slice *model.ItemType, parameter *Parameter, zone int, electromagneticStirringFactor float32) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2 int
	index1 = int(slice[y][x+1]) - 1
	index2 = int(slice[y-1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x+1, y, parameter, zone, electromagneticStirringFactor)/(stdXStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index2, x, y, x, y-1, parameter, zone, electromagneticStirringFactor)/(stdYStep*(getEy(y)+getEy(y-1))) +
		parameter.GetHeff(x, y, z)/(stdYStep)
	//fmt.Println("denominator", denominator, parameter.Density[index]*parameter.Enthalpy[index], "t: ", t)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case8 -> 左面边
func getDeltaTCase8(x, y int, slice *model.ItemType, parameter *Parameter, zone int, electromagneticStirringFactor float32) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2, index3 int
	index1 = int(slice[y][x+1]) - 1
	index2 = int(slice[y+1][x]) - 1
	index3 = int(slice[y-1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x+1, y, parameter, zone, electromagneticStirringFactor)/(stdXStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index2, x, y, x, y+1, parameter, zone, electromagneticStirringFactor)/(stdYStep*(getEy(y)+getEy(y+1))) +
		2*getLambda(index, index3, x, y, x, y-1, parameter, zone, electromagneticStirringFactor)/(stdYStep*(getEy(y)+getEy(y-1)))
	//fmt.Println("denominator", denominator, parameter.Density[index]*parameter.Enthalpy[index], "t: ", t)
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case9 -> 内部点
func getDeltaTCase9(x, y int, slice *model.ItemType, parameter *Parameter, zone int, electromagneticStirringFactor float32) float32 {
	var t = slice[y][x]
	var index = int(t) - 1
	var index1, index2, index3, index4 int
	index1 = int(slice[y][x-1]) - 1
	index2 = int(slice[y][x+1]) - 1
	index3 = int(slice[y+1][x]) - 1
	index4 = int(slice[y-1][x]) - 1
	denominator := 2*getLambda(index, index1, x, y, x-1, y, parameter, zone, electromagneticStirringFactor)/(stdXStep*(getEx(x)+getEx(x-1))) +
		2*getLambda(index, index2, x, y, x+1, y, parameter, zone, electromagneticStirringFactor)/(stdXStep*(getEx(x)+getEx(x+1))) +
		2*getLambda(index, index3, x, y, x, y+1, parameter, zone, electromagneticStirringFactor)/(stdYStep*(getEy(y)+getEy(y+1))) +
		2*getLambda(index, index4, x, y, x, y-1, parameter, zone, electromagneticStirringFactor)/(stdYStep*(getEy(y)+getEy(y-1)))
	return (parameter.Density[index] * parameter.Enthalpy[index]) / (t * denominator)
}

const bigNum = float32(3.0)

// 计算一个切片的时间步长
func calculateTimeStepOfOneSlice(z int, slice *model.ItemType, parameter *Parameter, zone int, electromagneticStirringFactor float32) float32 {
	// 计算时间步长 - start
	var deltaTArr = [9]float32{}
	deltaTArr[0] = getDeltaTCase1(0, 0, slice, parameter, zone, electromagneticStirringFactor)
	deltaTArr[1] = getDeltaTCase2(Length/XStep-2, 0, slice, parameter, zone, electromagneticStirringFactor)
	deltaTArr[2] = getDeltaTCase3(Length/XStep-1, 0, z, slice, parameter, zone, electromagneticStirringFactor)
	deltaTArr[3] = getDeltaTCase4(Length/XStep-1, Width/YStep-2, z, slice, parameter, zone, electromagneticStirringFactor)
	deltaTArr[4] = getDeltaTCase5(Length/XStep-1, Width/YStep-1, z, slice, parameter, zone, electromagneticStirringFactor)
	deltaTArr[5] = getDeltaTCase6(Length/XStep-2, Width/YStep-1, z, slice, parameter, zone, electromagneticStirringFactor)
	deltaTArr[6] = getDeltaTCase7(0, Width/YStep-1, z, slice, parameter, zone, electromagneticStirringFactor)
	deltaTArr[7] = getDeltaTCase8(0, Width/YStep-2, slice, parameter, zone, electromagneticStirringFactor)
	deltaTArr[8] = getDeltaTCase9(Length/XStep-2, Width/YStep-2, slice, parameter, zone, electromagneticStirringFactor)
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

func min(x, y float32) float32 {
	if x < y {
		return x
	}
	return y
}

func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

// 计算流速
func CalculateVwt(q, s float64) float64 {
	return 3000.0 / 1000.0 / 60.0 / s
}

// 32.22摄氏度
// ********************************[温度 0， 密度 1， 导热系数 2， 比热容 3， 运动粘度 4， 动力粘度 5， 普朗特数 6]
var parameterOfWaterLow = []float64{32.22, 994.9, 0.623, 4174.0, 0.0000007689, 0.000765, 5.13}
var parameterOfWaterHigh = []float64{37.78, 993.0, 0.630, 4174.0, 0.0000006868, 0.000682, 4.52}

// 计算冷却水在结晶器铜板冷却水道中产生的换热系数
func ROfWater(q, s float64, tempOfWater float64) float32 {
	Dh := 4 * (math.Pi*3 + 15 + 15 + 6) / (15*6 + math.Pi*9/2)
	Vwt := CalculateVwt(q, s)
	v := parameterOfWaterHigh[4] +
		(parameterOfWaterLow[4]-parameterOfWaterHigh[4])/
			(parameterOfWaterHigh[0]-parameterOfWaterLow[0])*
			(parameterOfWaterHigh[0]-tempOfWater) // 差值法计算运动粘度
	Pr := parameterOfWaterHigh[6] +
		(parameterOfWaterLow[6]-parameterOfWaterHigh[6])/
			(parameterOfWaterHigh[0]-parameterOfWaterLow[0])*
			(parameterOfWaterHigh[0]-tempOfWater) // 插值法计算普朗特数
	Red := Vwt * Dh / v                           // 雷诺数
	f := math.Pow(0.790*math.Log(Red)-1.64, -2)
	Nud := (f / 8) * (Red - 1000) * Pr / (1 + 12.7*math.Pow(f/8, 0.5)*(math.Pow(Pr, 2/3)-1))
	k := parameterOfWaterLow[2] +
		(parameterOfWaterHigh[2]-parameterOfWaterLow[2])/
			(parameterOfWaterHigh[0]-parameterOfWaterLow[0])*
			(tempOfWater-parameterOfWaterLow[0]) // 插值法计算导热系数
	return 1 / float32(k*Nud/Dh)
}

// 计算铜板的换热系数
func ROfCu() float32 {
	return 0.02 / 365.0
}

const (
	// 冷却介质
	Water       = 1
	WaterAndAir = 2
)

// 计算二冷区中的综合换热系数
func calculateAverageHeffHelper(L, AB, BC, CD, DE, Hbr float32, typ int, S, Volume, T, Ds, R0, Ts_ float64) float32 {
	// 计算三部分综合换热系数
	// L 表示辊间距离
	// typ 冷却介质类型
	// S 直接喷淋区域面积 m^2
	// Volume 水量
	// T 喷淋区域铸坯表面平均温度
	// Ds 喷水厚度
	// R0 辊子半径
	// Ts_ i-1号辊子处的铸坯表面温度
	// Hbr
	// 计算所需参数
	W := Volume / S // L/m2*s

	// 1. 直接喷淋区
	Hs := calculateHs(W, T, typ)

	// 2. 间接喷淋区
	Hs1 := calculateHs1(Ds, float64(L), Hs, Hbr)
	Hs2 := calculateHs2(Ds, float64(L), float64(DE), Hs, Hbr)

	// 3. 辊子直接接触区
	Hsr := calculateHsr(R0, float64(DE), Ts_)

	//fmt.Printf("L: %f, AB: %f, BC: %f, CD: %f, DE: %f, Hbr: %f, typ: %d, S: %f, Volume: %f, T: %f, Ds: %f, R0: %f, Ts_: %f\n", L, AB, BC, CD, DE, Hbr, typ, S, Volume, T, Ds, R0, Ts_)
	//fmt.Println("1. 直接喷淋区: ", Hs)
	//fmt.Println("2. 间接喷淋区: ", Hs1, Hs2)
	//fmt.Println("3. 辊子直接接触区: ", Hsr)
	return (Hs1*AB + Hs*BC + Hs2*CD + Hsr*DE) / L
}

// 1. 计算直接喷淋区域，目前使用该方法
func calculateHs(W, T float64, typ int) float32 {
	var a int
	var m, n float64
	if typ == Water {
		a = 1_900_000_000
		m = 0.660
		n = -2.290
	} else if typ == WaterAndAir {
		a = 822067
		m = 0.750
		n = -1.200
	}
	return float32(float64(a) * math.Pow(W, m) * math.Pow(T, n))
}

// 2. 计算AB, CD间平均换热系数
// Ds 直接喷淋厚度，Li为辊距
func calculateHs1(Ds, Li float64, Hs, Hbr float32) float32 {
	A1 := Ds / 2
	A2 := Li / 2
	Aa := A1 / A2
	return (Hs-Hbr)*float32(Aa+math.Pow(Aa, 2))/2 + Hbr
}

func calculateHs2(Ds, Li, DE float64, Hs, Hbr float32) float32 {
	A1 := Ds / 2
	A3 := Li/2 - DE
	Aa := A1 / A3
	return (Hs-Hbr)*float32(Aa+math.Pow(Aa, 2))/2 + Hbr
}

// 3. 计算辊子接触区域总换热系数Hsr
func calculateHsr(R0, DE, Ts_ float64) float32 {
	// R0 辊子半径
	// DE 铸坯与辊子间的接触长度
	// Ts_ i号辊子处的铸坯表面温度
	LambdaR := 3.489                    // 辊子材料的导热率
	As := 0.3489                        // 铸坯表面与辊子表面间的换热系数
	Aa := 1.9771 * 1 / 1000.0           // 辊子表面与大气间的换热系数
	Aw := 0.27726                       // 辊子内表面与冷却水之间的换热系数
	Ri := 3.8                           // 冷却水孔半径按 2 ~ 6cm计算,目前假定都是38mm
	As_ := DE / (2 * math.Pi * R0) * As // 铸坯到辊子表面换热系数转化成轴对称模型时的等价换热系数
	Ta := 50.0                          // 环境温度
	Tw := 20.0                          // 冷却水温度

	part1 := (math.Log(R0/Ri) + LambdaR/(Ri*Aw)) * (As_ + Aa*(Ta-Tw)/(Ts_-Tw))
	part2 := LambdaR*(1/R0+(As_+Aa)/(Ri*Aw)) + (As_+Aa)*math.Log(R0/Ri)

	return float32(As * (1 - part1/part2))
}

// 1. 计算直接喷淋区域，目前不使用该方法
func calculateHs_(typ int, D, B, Q float64, pre, cur int, Hbr float32) float32 {
	if typ == Water {
		return directAreaWater(D, B, Q, pre, cur, Hbr)
	} else if typ == WaterAndAir {
		return directAreaWaterAndGAir(D, B, Q, pre, cur, Hbr)
	}
	return 0
}

// D：喷淋厚度，B喷淋宽度，Q内弧侧水量
// 介质：纯水
func directAreaWater(D, B, Q float64, pre, cur int, Hbr float32) float32 {
	// 计算水量密度
	Vs := Q / (D * B * float64(cur-pre))
	// 计算Hs
	var Hs float64
	if Vs > 0.45 {
		Hs = 1.717372*math.Pow(Vs, 0.7) + 0.021
	} else {
		// todo 有疑问：公式确认

	}
	Hs_ := 1 / (1/Hs + 0.17197)
	return float32((Hs_-0.021)*0.8) + Hbr
}

// 介质：气水
func directAreaWaterAndGAir(D, B, Q float64, pre, cur int, Hbr float32) float32 {
	// 计算水量密度
	Vs := Q / (D * B * float64(cur-pre))
	// 计算Hs
	Hs := 1.488912*math.Pow(Vs, 0.75) + 0.021
	Hs_ := 1 / (1/Hs + 0.17197)
	return float32((Hs_-0.021)*0.8) + Hbr
}

// 计算空气换热系数
func calculateHbr(Ts_, Ta float64, parameter *Parameter) float32 {
	// Ts_: 上一个辊子处的铸坯表面温度
	// Ta: 环境温度
	ar := float64(parameter.Emissivity[int(Ts_)]) * 5.669 / (Ts_ - Ta) * (math.Pow((Ts_+273.0)/100, 4) - math.Pow((Ta+273.0)/100, 4))
	ac := 46.52
	return float32(ar + ac)
}

func calculateHbr_(Ts_, Ta float64) float32 {
	// Ts_: 上一个辊子处的铸坯表面温度
	// Ta: 环境温度
	ar := 0.807 * 5.669 / (Ts_ - Ta) * (math.Pow((Ts_+273.0)/100, 4) - math.Pow((Ta+273.0)/100, 4))
	ac := 46.52
	return float32(ar + ac)
}

// 计算空气换热系数
func calculateAc() float32 {
	return 46.52
}

// 计算辐射换热系数产生的热流密度
func calculateQar(Ts_, Ta float64) float32 {
	return float32(0.8 * 5.669 * (math.Pow((Ts_+273.0)/100, 4) - math.Pow((Ta+273.0)/100, 4)))
}

// 计算自然冷却区综合换热系数
func calculateHci(Hbr, Hsr, L, DE float32) float32 {
	return ((L - DE) * Hbr + Hsr * DE) / L
}

// 计算DE长度
func calculateDE(Droi, Drui, Deformation float64) float32 {
	// Droi 内弧辊子直径 Drui外弧辊子直径
	// Deformation鼓肚量
	var Dri float64 // 内外辊子平均直径
	Dri = 4 * Droi * Drui / math.Pow(math.Pow(Droi, 0.5)+math.Pow(Drui, 0.5), 2)

	return float32(0.6 * math.Pow(Dri*Deformation, 0.5))
}

// 计算鼓肚量
func calculateDeformation(centerRollersDistance, v, Hi float64, Si_1 float64, Tm, Tma float64) float64 {
	// Hi、Hi_1 代表辊子距离结晶器液面高度
	// D0 cm 铸坯厚度
	// Lwi 外弧线上辊距 cm
	// Ri 连铸机圆弧主半径 cm
	// v 拉速
	// Hi 第i个辊子处距结晶器液面的垂直高度
	// Hi_1 第i-1个辊子处距结晶器液面的垂直高度
	// Si_1 前一个辊子处坯壳厚度
	// Tm 凝固温度，用液相线温度
	// Tma 铸坯坯壳平均温度，Tma = (Tm + Tsi_1) / 2。Tsi_1为上一个辊子处铸坯表面的温度
	Pi := 0.1 * 7.0 * Hi
	E := (Tm - Tma) / (Tm - 100) * 10000.0
	ts := centerRollersDistance / v // min
	//fmt.Printf("Pi: %f, E: %f, Ts: %f, Hi: %f, Si_1: %f, Tm: %f, Tma: %f\n", Pi, E, ts, Hi, Si_1, Tm, Tma)
	//fmt.Println("鼓肚量为：", Pi*math.Pow(centerRollersDistance, 4)*math.Pow(ts, 0.5)/(32*E*math.Pow(Si_1, 3)))
	return Pi * math.Pow(centerRollersDistance, 4) * math.Pow(ts, 0.5) / (32 * E * math.Pow(Si_1, 3))
}

func calculateSolidFraction(T, Ts, Tl float32) float32 {
	//member := (Ts - T) + (2 / math.Pi) * (Ts - Tl) * (1 - float32(math.Cos(float64(math.Pi / 2 * (T - Tl) / (Ts - Tl)))))
	//denominator := (Tl - Ts) * (1 - math.Pi / 2)
	//return member / denominator
	return (Tl - T) / (Tl - Ts)
}

func handleData(data []byte) {
	var NozzleCfg = model.NozzleCfg{
		WideItems:   make([]model.WideItem, 0),
		NarrowItems: make([]model.NarrowItem, 0),
	}
	err := json.Unmarshal(data, &NozzleCfg)
	if err != nil {
		return
	}
	pre := float32(850.0)
	for i := 0; i < len(NozzleCfg.WideItems)-1; i++ {
		cur := NozzleCfg.WideItems[i].Distance
		NozzleCfg.WideItems[i].RollerDistance = cur - pre
		pre = cur
	}
	// 2 - 4
	alter := 1
	for i := 2; i < 20; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -650.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 650.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -530.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 530.0
		}
		alter ^= 1
	}
	// 5
	alter = 1
	for i := 20; i < 24; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -583.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 583.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -475.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 475.0
		}
		alter ^= 1
	}
	alter = 0
	for i := 24; i < 27; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -583.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 583.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -475.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 475.0
		}
		alter ^= 1
	}
	// 6
	alter = 1
	for i := 27; i < 31; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -583.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 583.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -475.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 475.0
		}
		alter ^= 1
	}
	alter = 0
	for i := 31; i < 34; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -583.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 583.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -475.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 475.0
		}
		alter ^= 1
	}
	// 7
	alter = 1
	for i := 34; i < 38; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -583.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 583.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -475.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 475.0
		}
		alter ^= 1
	}
	alter = 0
	for i := 38; i < 41; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -583.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 583.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -475.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 475.0
		}
		alter ^= 1
	}
	alter = 1
	for i := 41; i < 45; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -583.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 583.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -475.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 475.0
		}
		alter ^= 1
	}
	alter = 0
	for i := 45; i < 48; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -583.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 583.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -475.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 475.0
		}
		alter ^= 1
	}
	// 8
	alter = 1
	for i := 48; i < 52; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -456.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 456.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -564.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 564.0
		}
		alter ^= 1
	}
	alter = 0
	for i := 52; i < 55; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -456.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 456.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -564.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 564.0
		}
		alter ^= 1
	}
	alter = 1
	for i := 55; i < 59; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -456.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 456.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -564.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 564.0
		}
		alter ^= 1
	}
	alter = 0
	for i := 59; i < 62; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -456.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 456.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -564.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 564.0
		}
		alter ^= 1
	}
	// 9
	alter = 1
	for i := 62; i < 66; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -456.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 456.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -564.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 564.0
		}
		alter ^= 1
	}
	alter = 0
	for i := 66; i < 69; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -456.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 456.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -564.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 564.0
		}
		alter ^= 1
	}
	alter = 1
	for i := 69; i < 73; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -456.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 456.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -564.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 564.0
		}
		alter ^= 1
	}
	alter = 0
	for i := 73; i < 76; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -456.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 456.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -564.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 564.0
		}
		alter ^= 1
	}
	// 10
	alter = 1
	for i := 76; i < 80; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -456.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 456.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -564.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 564.0
		}
		alter ^= 1
	}
	alter = 0
	for i := 80; i < 83; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -456.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 456.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -564.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 564.0
		}
		alter ^= 1
	}
	alter = 1
	for i := 83; i < 87; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -456.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 456.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -564.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 564.0
		}
		alter ^= 1
	}
	alter = 0
	for i := 87; i < 90; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -456.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 456.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -564.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 564.0
		}
		alter ^= 1
	}
	alter = 1
	for i := 90; i < 94; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -456.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 456.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -564.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 564.0
		}
		alter ^= 1
	}
	alter = 0
	for i := 94; i < 97; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -456.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 456.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -564.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 564.0
		}
		alter ^= 1
	}
	// 11
	alter = 1
	for i := 97; i < 101; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -456.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 456.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -564.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 564.0
		}
		alter ^= 1
	}
	alter = 0
	for i := 101; i < 104; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -456.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 456.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -564.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 564.0
		}
		alter ^= 1
	}
	alter = 1
	for i := 104; i < 107; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -456.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 456.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -564.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 564.0
		}
		alter ^= 1
	}
	alter = 0
	for i := 107; i < 111; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -456.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 456.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -564.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 564.0
		}
		alter ^= 1
	}
	alter = 1
	for i := 111; i < 115; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -456.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 456.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -564.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 564.0
		}
		alter ^= 1
	}
	alter = 0
	for i := 115; i < 118; i++ {
		if alter == 1 {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -456.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 456.0
		} else {
			NozzleCfg.WideItems[i].CenterSpraySection.LeftLimit = -564.0
			NozzleCfg.WideItems[i].CenterSpraySection.RightLimit = 564.0
		}
		alter ^= 1
	}
	data, err = json.Marshal(NozzleCfg)
	if err != nil {
		return
	}
	err = ioutil.WriteFile("E:/GoWorkPlace/src/lz/conf/generate.json", data, 0644)
	if err != nil {
		return
	}
}
