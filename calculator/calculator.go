package calculator

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// 步长设定
// 1. x方向（宽边方向）5mm
// 2. y方向（窄边方向）5mm
// 3. z方向（拉坯方向），板坯切片厚度方向5mm / 10mm

// 参数解释
// 1. 从节点[i, j] 到 [x, y] 实际等效导热系数 lambda
// 2. 每个节点的密度，density
// 3. 边界节点热流密度，Q
// 4. 边界节点综合换热系数，heff

// 全局变量
// 步长 和 铸坯的尺寸，单位mm

const (
	XStep       = 5
	YStep       = 5
	ZStep       = 10
	Length      = 2700 / 2
	Width       = 420 / 2
	ZLength     = 40000
	ArrayLength = 1550 / 5
)

var (
	Density  [ArrayLength]float32 // 密度
	Enthalpy [ArrayLength]float32 // 焓
	Lambda   [ArrayLength]float32 // 导热系数
	HEff     [ArrayLength]float32 // 综合换热系数, 注意：简单处理了！
	Q        [ArrayLength]float32 // 热流密度, 注意：简单处理了
	C        [ArrayLength]float32 // 比热容

	ThermalFieldPtr  *[ZLength / ZStep][Width / YStep][Length / XStep]float32
	ThermalField1Ptr *[ZLength / ZStep][Width / YStep][Length / XStep]float32

	ThermalField  [ZLength / ZStep][Width / YStep][Length / XStep]float32
	ThermalField1 [ZLength / ZStep][Width / YStep][Length / XStep]float32

	// 铸机满了之后，新加入的切片先放入的三维数组，满了之后交换指针指向
	ThermalFieldCopyPtr ThermalFieldStruct
	ThermalFieldCopy    [ZLength / ZStep][Width / YStep][Length / XStep]float32 // 存放三维数组满了之后，新进入的切片
)

type ThermalFieldStruct struct {
	Start  int
	End    int
	Field  *[ZLength / ZStep][Width / YStep][Length / XStep]float32
	IsFull bool
	IsTail bool
	IsCopy bool
}

type Calculator struct {
	// 计算参数
	EdgeWidth int

	Step int // 当c.EdgeWidth > 0, step = 2;

	initialTemperature float32
	ThermalField       ThermalFieldStruct

	// 每计算一个 ▲t 进行一次异或运算
	alternating bool

	//v int 拉速
	V        int64
	reminder int64

	CalcHub *CalcHub

	Mu sync.Mutex // 保护push data时对温度数据的并发访问
}

func NewCalculator(edgeWidth int) *Calculator {
	// 0 <= edgeWidth <= 20
	if edgeWidth < 0 {
		edgeWidth = 0
	}
	if edgeWidth > 20 {
		edgeWidth = 20
	}
	calculator := Calculator{}
	// 方程初始条件为 T = Tw，Tw为钢水刚到弯月面处的温度。

	// 对于1/4模型，如果不考虑沿着拉坯方向的传热，则每个切片是首切片、中间切片和尾切片均相同，
	// 仅需要图中的四个角部短点、四个边界节点和内部节点的不同，给出9种不同位置的差分方程。
	// 初始化
	// 1. 初始化网格划分的各个节点的初始温度
	var start = time.Now()

	// 初始化温度场
	calculator.initialTemperature = 1550.0
	for z := 0; z < ZLength/ZStep; z++ {
		for y := 0; y < Width/YStep; y++ {
			for x := 0; x < Length/XStep; x++ {
				ThermalField[z][y][x] = calculator.initialTemperature
				ThermalField1[z][y][x] = calculator.initialTemperature
			}
		}
	}
	ThermalFieldPtr = &ThermalField
	ThermalField1Ptr = &ThermalField1
	calculator.ThermalField.Start = ZLength / ZStep
	calculator.ThermalField.End = ZLength / ZStep
	calculator.ThermalField.Field = &ThermalField

	// 2. 导热系数，200℃ 到 1600℃，随温度的上升而下降
	var LambdaStart = float32(500.0)
	var LambdaIter = float32(50.0-45.0) / ArrayLength
	for i := 0; i < ArrayLength; i++ {
		Lambda[i] = LambdaStart - float32(i)*LambdaIter
	}

	// 3. 密度
	var DensityStart = float32(8.0)
	var DensityIter = float32(8.0-7.0) / ArrayLength
	for i := 0; i < ArrayLength; i++ {
		Density[i] = DensityStart - float32(i)*DensityIter
	}

	// 4. 焓值
	var EnthalpyStart = float32(1000.0)
	var EnthalpyStep = float32(100) / ArrayLength
	for i := 0; i < ArrayLength; i++ {
		Enthalpy[i] = EnthalpyStart + float32(i)*EnthalpyStep
	}

	// 5. 综合换热系数
	var HEffStart = float32(5.0)
	var HEffStep = float32(20.0-15.0) / ArrayLength
	for i := 0; i < ArrayLength; i++ {
		HEff[i] = HEffStart + float32(i)*HEffStep
	}

	// 6. 热流密度
	var QStart = float32(4000.0)
	var QStep = float32(5000.0) / ArrayLength
	for i := 0; i < ArrayLength; i++ {
		Q[i] = QStart + float32(i)*QStep
	}

	// 7. 比热容
	var CStart = float32(10.0)
	var CStep = float32(1.0) / ArrayLength
	for i := 0; i < ArrayLength; i++ {
		C[i] = CStart + float32(i)*CStep
	}

	calculator.EdgeWidth = edgeWidth
	calculator.Step = 1
	if calculator.EdgeWidth > 0 {
		calculator.Step = 2
	}

	calculator.V = int64(1.5 * 1000 / 60) // m / min
	calculator.alternating = true
	calculator.CalcHub = NewCalcHub()

	ThermalFieldCopyPtr.Start = ZLength / ZStep
	ThermalFieldCopyPtr.End = ZLength / ZStep
	ThermalFieldCopyPtr.Field = &ThermalFieldCopy
	ThermalFieldCopyPtr.IsCopy = true
	fmt.Println("初始化时间: ", time.Since(start))
	return &calculator
}

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
func (c *Calculator) GetLambda(index1, index2, x1, y1, x2, y2 int) float32 {
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

// 计算时间步长 case1 -> 左下角
func (c *Calculator) GetDeltaTCase1(z, x, y int, Field *[ZLength / ZStep][Width / YStep][Length / XStep]float32) float32 {
	var t = Field[z][y][x]
	var index = int(t)/5 - 1
	var index1, index2 int
	index1 = int(Field[z][y][x+1])/5 - 1
	index2 = int(Field[z][y+1][x])/5 - 1
	denominator := 2*c.GetLambda(index, index1, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
		2*c.GetLambda(index, index2, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1)))
	return (Density[index] * Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case2 -> 下面边
func (c *Calculator) GetDeltaTCase2(z, x, y int, Field *[ZLength / ZStep][Width / YStep][Length / XStep]float32) float32 {
	var t = Field[z][y][x]
	var index = int(t)/5 - 1
	var index1, index2, index3 int
	index1 = int(Field[z][y][x-1])/5 - 1
	index2 = int(Field[z][y][x+1])/5 - 1
	index3 = int(Field[z][y+1][x])/5 - 1
	denominator := 2*c.GetLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
		2*c.GetLambda(index, index2, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
		2*c.GetLambda(index, index3, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1)))
	return (Density[index] * Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case3 -> 右下角
func (c *Calculator) GetDeltaTCase3(z, x, y int, Field *[ZLength / ZStep][Width / YStep][Length / XStep]float32) float32 {
	var t = Field[z][y][x]
	var index = int(t)/5 - 1
	var index1, index2 int
	index1 = int(Field[z][y][x-1])/5 - 1
	index2 = int(Field[z][y+1][x])/5 - 1
	denominator := 2*c.GetLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
		2*c.GetLambda(index, index2, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1))) +
		HEff[index]/(XStep)
	return (Density[index] * Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case4 -> 右面边
func (c *Calculator) GetDeltaTCase4(z, x, y int, Field *[ZLength / ZStep][Width / YStep][Length / XStep]float32) float32 {
	var t = Field[z][y][x]
	var index = int(t)/5 - 1
	var index1, index2, index3 int
	index1 = int(Field[z][y][x-1])/5 - 1
	index2 = int(Field[z][y+1][x])/5 - 1
	index3 = int(Field[z][y-1][x])/5 - 1
	denominator := 2*c.GetLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
		2*c.GetLambda(index, index2, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1))) +
		2*c.GetLambda(index, index3, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1))) +
		HEff[index]/(XStep)
	return (Density[index] * Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case5 -> 右上角
func (c *Calculator) GetDeltaTCase5(z, x, y int, Field *[ZLength / ZStep][Width / YStep][Length / XStep]float32) float32 {
	var t = Field[z][y][x]
	var index = int(t)/5 - 1
	var index1, index2 int
	index1 = int(Field[z][y][x-1])/5 - 1
	index2 = int(Field[z][y-1][x])/5 - 1
	denominator := 2*c.GetLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
		2*c.GetLambda(index, index2, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1))) +
		HEff[index]/(XStep) +
		HEff[index]/(YStep)
	return (Density[index] * Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case6 -> 上面边
func (c *Calculator) GetDeltaTCase6(z, x, y int, Field *[ZLength / ZStep][Width / YStep][Length / XStep]float32) float32 {
	var t = Field[z][y][x]
	var index = int(t)/5 - 1
	var index1, index2, index3 int
	index1 = int(Field[z][y][x-1])/5 - 1
	index2 = int(Field[z][y][x+1])/5 - 1
	index3 = int(Field[z][y-1][x])/5 - 1
	denominator := 2*c.GetLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
		2*c.GetLambda(index, index2, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
		2*c.GetLambda(index, index3, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1))) +
		HEff[index]/(YStep)
	return (Density[index] * Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case7 -> 左上角
func (c *Calculator) GetDeltaTCase7(z, x, y int, Field *[ZLength / ZStep][Width / YStep][Length / XStep]float32) float32 {
	var t = Field[z][y][x]
	var index = int(t)/5 - 1
	var index1, index2 int
	index1 = int(Field[z][y][x+1])/5 - 1
	index2 = int(Field[z][y-1][x])/5 - 1
	denominator := 2*c.GetLambda(index, index1, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
		2*c.GetLambda(index, index2, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1))) +
		HEff[index]/(YStep)
	return (Density[index] * Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case8 -> 左面边
func (c *Calculator) GetDeltaTCase8(z, x, y int, Field *[ZLength / ZStep][Width / YStep][Length / XStep]float32) float32 {
	var t = Field[z][y][x]
	var index = int(t)/5 - 1
	var index1, index2, index3 int
	index1 = int(Field[z][y][x+1])/5 - 1
	index2 = int(Field[z][y+1][x])/5 - 1
	index3 = int(Field[z][y-1][x])/5 - 1
	denominator := 2*c.GetLambda(index, index1, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
		2*c.GetLambda(index, index2, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1))) +
		2*c.GetLambda(index, index3, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1)))
	return (Density[index] * Enthalpy[index]) / (t * denominator)
}

// 计算时间步长 case9 -> 内部点
func (c *Calculator) GetDeltaTCase9(z, x, y int, Field *[ZLength / ZStep][Width / YStep][Length / XStep]float32) float32 {
	var t = Field[z][y][x]
	var index = int(t)/5 - 1
	var index1, index2, index3, index4 int
	index1 = int(Field[z][y][x-1])/5 - 1
	index2 = int(Field[z][y][x+1])/5 - 1
	index3 = int(Field[z][y+1][x])/5 - 1
	index4 = int(Field[z][y-1][x])/5 - 1
	denominator := 2*c.GetLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
		2*c.GetLambda(index, index2, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
		2*c.GetLambda(index, index3, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1))) +
		2*c.GetLambda(index, index4, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1)))
	return (Density[index] * Enthalpy[index]) / (t * denominator)
}

// 计算一个切片的时间步长
func (c *Calculator) calculateTimeStepOfOneSlice(z int, Field *[ZLength / ZStep][Width / YStep][Length / XStep]float32) float32 {
	// 计算时间步长 - start
	var deltaTArr = [9]float32{}
	deltaTArr[0] = c.GetDeltaTCase1(z, 0, 0, Field)
	deltaTArr[1] = c.GetDeltaTCase2(z, Length/XStep-2, 0, Field)
	deltaTArr[2] = c.GetDeltaTCase3(z, Length/XStep-1, 0, Field)
	deltaTArr[3] = c.GetDeltaTCase4(z, Length/XStep-1, Width/YStep-2, Field)
	deltaTArr[4] = c.GetDeltaTCase5(z, Length/XStep-1, Width/YStep-1, Field)
	deltaTArr[5] = c.GetDeltaTCase6(z, Length/XStep-2, Width/YStep-1, Field)
	deltaTArr[6] = c.GetDeltaTCase7(z, 0, Width/YStep-1, Field)
	deltaTArr[7] = c.GetDeltaTCase8(z, 0, Width/YStep-2, Field)
	deltaTArr[8] = c.GetDeltaTCase9(z, Length/XStep-2, Width/YStep-2, Field)
	var min = float32(1000.0) // 模拟一个很大的数
	for _, i := range deltaTArr {
		if min > i {
			min = i
		}
	}
	return min
	// 计算时间步长 - end
}

// 计算所有切片中最短的时间步长
func (c *Calculator) calculateTimeStep() (float32, time.Duration) {
	start := time.Now()
	min := float32(1000.0)
	var t float32
	for z := ThermalFieldCopyPtr.Start; z < ThermalFieldCopyPtr.End; z++ {
		t = c.calculateTimeStepOfOneSlice(z, ThermalFieldCopyPtr.Field)
		if t < min {
			min = t
		}
	}
	for z := c.ThermalField.Start; z < c.ThermalField.End; z++ {
		t = c.calculateTimeStepOfOneSlice(z, c.ThermalField.Field)
		if t < min {
			min = t
		}
	}
	//fmt.Println("计算deltaT花费的时间：", time.Since(start), min)
	return min, time.Since(start)
}

// 计算一个left top点的温度变化
func (c *Calculator) calculatePointLT(deltaT float32, z int, ThermalField ThermalFieldStruct) {
	Field := ThermalField.Field
	var index = int(Field[z][Width/YStep-1][0])/5 - 1
	var index1 = int(Field[z][Width/YStep-1][1])/5 - 1
	var index2 = int(Field[z][Width/YStep-2][0])/5 - 1
	var deltaHlt = c.GetLambda(index, index1, 0, Width/YStep-1, 1, Width/YStep-1)*(Field[z][Width/YStep-1][0]-Field[z][Width/YStep-1][1])/float32(XStep*(getEx(1)+getEx(0))) +
		c.GetLambda(index, index2, 0, Width/YStep-1, 0, Width/YStep-2)*(Field[z][Width/YStep-1][0]-Field[z][Width/YStep-2][0])/float32(YStep*(getEy(Width/YStep-2)+getEy(Width/YStep-1))) +
		Q[index]/(2*YStep)

	deltaHlt = deltaHlt * (2 * deltaT / Density[index])
	//fmt.Println(ThermalField[z][Width/YStep-1][0]-ThermalField[z][Width/YStep-1][1], ThermalField[z][Width/YStep-1][0]-ThermalField[z][Width/YStep-2][0], Q[index], deltaHlt/C[index], "左上角")
	k := z
	if ThermalField.IsFull {
		k = z
		if ThermalField.IsCopy {
			k = (ThermalField.End - ThermalField.Start) - (ThermalField.End - z)
		}
	}
	if c.alternating {
		ThermalField1Ptr[k][Width/YStep-1][0] = Field[z][Width/YStep-1][0] - deltaHlt/C[index]
	} else {
		// 需要修改焓的变化到温度变化k映射关系
		ThermalFieldPtr[k][Width/YStep-1][0] = Field[z][Width/YStep-1][0] - (deltaHlt / C[index])
	}
}

// 计算上表面点温度变化
func (c *Calculator) calculatePointTA(deltaT float32, x, z int, ThermalField ThermalFieldStruct) {
	Field := ThermalField.Field
	var index = int(Field[z][Width/YStep-1][x])/5 - 1
	var index1 = int(Field[z][Width/YStep-1][x-1])/5 - 1
	var index2 = int(Field[z][Width/YStep-1][x+1])/5 - 1
	var index3 = int(Field[z][Width/YStep-2][x])/5 - 1
	var deltaHta = c.GetLambda(index, index1, x, Width/YStep-1, x-1, Width/YStep-1)*(Field[z][Width/YStep-1][x]-Field[z][Width/YStep-1][x-1])/float32(XStep*(getEx(x-1)+getEx(x))) +
		c.GetLambda(index, index2, x, Width/YStep-1, x+1, Width/YStep-1)*(Field[z][Width/YStep-1][x]-Field[z][Width/YStep-1][x+1])/float32(XStep*(getEx(x)+getEx(x+1))) +
		c.GetLambda(index, index3, x, Width/YStep-1, x, Width/YStep-2)*(Field[z][Width/YStep-1][x]-Field[z][Width/YStep-2][x])/float32(YStep*(getEy(Width/YStep-2)+getEy(Width/YStep-1))) +
		Q[index]/(2*YStep)

	deltaHta = deltaHta * (2 * deltaT / Density[index])
	//fmt.Println(ThermalField[z][Width/YStep-1][x]-ThermalField[z][Width/YStep-1][x-1], ThermalField[z][Width/YStep-1][x]-ThermalField[z][Width/YStep-1][x+1], ThermalField[z][Width/YStep-1][x]-ThermalField[z][Width/YStep-2][x], Q[index], deltaHta/C[index], "上表面")
	k := z
	if ThermalField.IsFull {
		k = z
		if ThermalField.IsCopy {
			k = (ThermalField.End - ThermalField.Start) - (ThermalField.End - z)
		}
	}
	if c.alternating {
		ThermalField1Ptr[k][Width/YStep-1][x] = Field[z][Width/YStep-1][x] - deltaHta/C[index]
	} else {
		// 需要修改焓的变化到温度变化k映射关系
		ThermalFieldPtr[k][Width/YStep-1][x] = Field[z][Width/YStep-1][x] - deltaHta/C[index]
	}
}

// 计算right top点的温度变化
func (c *Calculator) calculatePointRT(deltaT float32, z int, ThermalField ThermalFieldStruct) {
	Field := ThermalField.Field
	var index = int(Field[z][Width/YStep-1][Length/XStep-1])/5 - 1
	var index1 = int(Field[z][Width/YStep-1][Length/XStep-2])/5 - 1
	var index2 = int(Field[z][Width/YStep-2][Length/XStep-1])/5 - 1
	var deltaHrt = c.GetLambda(index, index1, Length/XStep-1, Width/YStep-1, Length/XStep-2, Width/YStep-1)*(Field[z][Width/YStep-1][Length/XStep-1]-Field[z][Width/YStep-1][Length/XStep-2])/float32(XStep*(getEx(Length/XStep-2)+getEx(Length/XStep-1))) +
		c.GetLambda(index, index2, Length/XStep-1, Width/YStep-1, Length/XStep-1, Width/YStep-2)*(Field[z][Width/YStep-1][Length/XStep-1]-Field[z][Width/YStep-2][Length/XStep-1])/float32(YStep*(getEy(Width/YStep-2)+getEy(Width/YStep-1))) +
		Q[index]/(2*YStep) +
		Q[index]/(2*XStep)

	deltaHrt = deltaHrt * (2 * deltaT / Density[index])
	//fmt.Println(ThermalField[z][Width/YStep-1][Length/XStep-1]-ThermalField[z][Width/YStep-1][Length/XStep-2], ThermalField[z][Width/YStep-1][Length/XStep-1]-ThermalField[z][Width/YStep-2][Length/XStep-1], Q[index], deltaHrt/C[index],  "右上角")
	k := z
	if ThermalField.IsFull {
		k = z
		if ThermalField.IsCopy {
			k = (ThermalField.End - ThermalField.Start) - (ThermalField.End - z)
		}
	}
	if c.alternating {
		ThermalField1Ptr[k][Width/YStep-1][Length/XStep-1] = Field[z][Width/YStep-1][Length/XStep-1] - deltaHrt/C[index] // 需要修改焓的变化到温度变化的映射关系
	} else {
		ThermalFieldPtr[k][Width/YStep-1][Length/XStep-1] = Field[z][Width/YStep-1][Length/XStep-1] - deltaHrt/C[index]
	}
}

// 计算右表面点的温度变化
func (c *Calculator) calculatePointRA(deltaT float32, y, z int, ThermalField ThermalFieldStruct) {
	Field := ThermalField.Field
	var index = int(Field[z][y][Length/XStep-1])/5 - 1
	var index1 = int(Field[z][y][Length/XStep-2])/5 - 1
	var index2 = int(Field[z][y-1][Length/XStep-1])/5 - 1
	var index3 = int(Field[z][y+1][Length/XStep-1])/5 - 1
	var deltaHra = c.GetLambda(index, index1, Length/XStep-1, y, Length/XStep-2, y)*(Field[z][y][Length/XStep-1]-Field[z][y][Length/XStep-2])/float32(XStep*(getEx(Length/XStep-2)+getEx(Length/XStep-1))) +
		c.GetLambda(index, index2, Length/XStep-1, y, Length/XStep-1, y-1)*(Field[z][y][Length/XStep-1]-Field[z][y-1][Length/XStep-1])/float32(YStep*(getEy(y-1)+getEy(y))) +
		c.GetLambda(index, index3, Length/XStep-1, y, Length/XStep-1, y+1)*(Field[z][y][Length/XStep-1]-Field[z][y+1][Length/XStep-1])/float32(YStep*(getEy(y+1)+getEy(y))) +
		Q[index]/(2*XStep)

	deltaHra = deltaHra * (2 * deltaT / Density[index])
	//fmt.Println(ThermalField[z][y][Length/XStep-1]-ThermalField[z][y][Length/XStep-2], ThermalField[z][y][Length/XStep-1]-ThermalField[z][y-1][Length/XStep-1], ThermalField[z][y][Length/XStep-1]-ThermalField[z][y+1][Length/XStep-1], Q[index], deltaHra/C[index], "右表面")
	k := z
	if ThermalField.IsFull {
		k = z
		if ThermalField.IsCopy {
			k = (ThermalField.End - ThermalField.Start) - (ThermalField.End - z)
		}
	}
	if c.alternating {
		ThermalField1Ptr[k][y][Length/XStep-1] = Field[z][y][Length/XStep-1] - deltaHra/C[index]
		// 需要修改焓的变化到温度变化的映射关系
	} else {
		ThermalFieldPtr[k][y][Length/XStep-1] = Field[z][y][Length/XStep-1] - deltaHra/C[index]
	}
}

// 计算right bottom点的温度变化
func (c *Calculator) calculatePointRB(deltaT float32, z int, ThermalField ThermalFieldStruct) {
	Field := ThermalField.Field
	var index = int(Field[z][0][Length/XStep-1])/5 - 1
	var index1 = int(Field[z][0][Length/XStep-2])/5 - 1
	var index2 = int(Field[z][1][Length/XStep-1])/5 - 1
	var deltaHrb = c.GetLambda(index, index1, Length/XStep-1, 0, Length/XStep-2, 0)*(Field[z][0][Length/XStep-1]-Field[z][0][Length/XStep-2])/float32(XStep*(getEx(Length/XStep-2)+getEx(Length/XStep-1))) +
		c.GetLambda(index, index2, Length/XStep-1, 0, Length/XStep-1, 1)*(Field[z][0][Length/XStep-1]-Field[z][1][Length/XStep-1])/float32(YStep*(getEy(1)+getEy(0))) +
		Q[index]/(2*XStep)

	deltaHrb = deltaHrb * (2 * deltaT / Density[index])
	//fmt.Println(ThermalField[z][0][Length/XStep-1]-ThermalField[z][0][Length/XStep-2], ThermalField[z][0][Length/XStep-1]-ThermalField[z][1][Length/XStep-1], Q[index],deltaHrb/C[index], "右下角")
	k := z
	if ThermalField.IsFull {
		k = z
		if ThermalField.IsCopy {
			k = (ThermalField.End - ThermalField.Start) - (ThermalField.End - z)
		}
	}
	if c.alternating {
		ThermalField1Ptr[k][0][Length/XStep-1] = Field[z][0][Length/XStep-1] - deltaHrb/C[index] // 需要修改焓的变化到温度变化的映射关系
	} else {
		ThermalFieldPtr[k][0][Length/XStep-1] = Field[z][0][Length/XStep-1] - deltaHrb/C[index]
	}
}

// 计算下表面点的温度变化
func (c *Calculator) calculatePointBA(deltaT float32, x, z int, ThermalField ThermalFieldStruct) {
	Field := ThermalField.Field
	var index = int(Field[z][0][x])/5 - 1
	var index1 = int(Field[z][0][x-1])/5 - 1
	var index2 = int(Field[z][0][x+1])/5 - 1
	var index3 = int(Field[z][1][x])/5 - 1
	var deltaHba = c.GetLambda(index, index1, x, 0, x-1, 0)*(Field[z][0][x]-Field[z][0][x-1])/float32(XStep*(getEx(x-1)+getEx(x))) +
		c.GetLambda(index, index2, x, 0, x+1, 0)*(Field[z][0][x]-Field[z][0][x+1])/float32(XStep*(getEx(x+1)+getEx(x))) +
		c.GetLambda(index, index3, x, 0, x, 1)*(Field[z][0][x]-Field[z][1][x])/float32(YStep*(getEy(1)+getEy(0)))

	deltaHba = deltaHba * (2 * deltaT / Density[index])
	//fmt.Println(ThermalField[z][0][x]-ThermalField[z][0][x-1], ThermalField[z][0][x]-ThermalField[z][0][x+1], ThermalField[z][0][x]-ThermalField[z][1][x],deltaHba/C[index], "下表面")
	k := z
	if ThermalField.IsFull {
		k = z
		if ThermalField.IsCopy {
			k = (ThermalField.End - ThermalField.Start) - (ThermalField.End - z)
		}
	}
	if c.alternating {
		ThermalField1Ptr[k][0][x] = Field[z][0][x] - deltaHba/C[index] // 需要修改焓的变化到温度变化的映射关系
	} else {
		ThermalFieldPtr[k][0][x] = Field[z][0][x] - deltaHba/C[index]
	}
}

// 计算left bottom点的温度变化
func (c *Calculator) calculatePointLB(deltaT float32, z int, ThermalField ThermalFieldStruct) {
	Field := ThermalField.Field
	var index = int(Field[z][0][0])/5 - 1
	var index1 = int(Field[z][0][1])/5 - 1
	var index2 = int(Field[z][1][0])/5 - 1
	var deltaHlb = c.GetLambda(index, index1, 1, 0, 0, 0)*(Field[z][0][0]-Field[z][0][1])/float32(XStep*(getEx(0)+getEx(1))) +
		c.GetLambda(index, index2, 0, 1, 0, 0)*(Field[z][0][0]-Field[z][1][0])/float32(YStep*(getEy(1)+getEy(0)))

	deltaHlb = deltaHlb * (2 * deltaT / Density[index])
	//fmt.Println(ThermalField[z][0][0]-ThermalField[z][0][1], ThermalField[z][0][0]-ThermalField[z][1][0],deltaHlb/C[index], "左下角")
	k := z
	if ThermalField.IsFull {
		k = z
		if ThermalField.IsCopy {
			k = (ThermalField.End - ThermalField.Start) - (ThermalField.End - z)
		}
	}
	if c.alternating {
		ThermalField1Ptr[k][0][0] = Field[z][0][0] - deltaHlb/C[index] // 需要修改焓的变化到温度变化的映射关系
	} else {
		ThermalFieldPtr[k][0][0] = Field[z][0][0] - deltaHlb/C[index]
	}
}

// 计算左表面点温度的变化
func (c *Calculator) calculatePointLA(deltaT float32, y, z int, ThermalField ThermalFieldStruct) {
	Field := ThermalField.Field
	var index = int(Field[z][y][0])/5 - 1
	var index1 = int(Field[z][y][1])/5 - 1
	var index2 = int(Field[z][y-1][0])/5 - 1
	var index3 = int(Field[z][y+1][0])/5 - 1
	var deltaHla = c.GetLambda(index, index1, 1, y, 0, y)*(Field[z][y][0]-Field[z][y][1])/float32(XStep*(getEx(0)+getEx(1))) +
		c.GetLambda(index, index2, 0, y-1, 0, y)*(Field[z][y][0]-Field[z][y-1][0])/float32(YStep*(getEy(y)+getEy(y-1))) +
		c.GetLambda(index, index3, 0, y+1, 0, y)*(Field[z][y][0]-Field[z][y+1][0])/float32(YStep*(getEy(y)+getEy(y+1)))
	deltaHla = deltaHla * (2 * deltaT / Density[index])
	//fmt.Println(ThermalField[z][y][0]-ThermalField[z][y][1], ThermalField[z][y][0]-ThermalField[z][y-1][0], ThermalField[z][y][0]-ThermalField[z][y+1][0], deltaHla/C[index], "左表面")
	k := z
	if ThermalField.IsFull {
		k = z
		if ThermalField.IsCopy {
			k = (ThermalField.End - ThermalField.Start) - (ThermalField.End - z)
		}
	}
	if c.alternating {
		ThermalField1Ptr[k][y][0] = Field[z][y][0] - deltaHla/C[index] // 需要修改焓的变化到温度变化的映射关系
	} else {
		ThermalFieldPtr[k][y][0] = Field[z][y][0] - deltaHla/C[index]
	}
}

// 计算内部点的温度变化
func (c *Calculator) calculatePointIN(deltaT float32, x, y, z int, ThermalField ThermalFieldStruct) {
	Field := ThermalField.Field
	var index = int(Field[z][y][x])/5 - 1
	var index1 = int(Field[z][y][x-1])/5 - 1
	var index2 = int(Field[z][y][x+1])/5 - 1
	var index3 = int(Field[z][y-1][x])/5 - 1
	var index4 = int(Field[z][y+1][x])/5 - 1
	var deltaHin = c.GetLambda(index, index1, x-1, y, x, y)*(Field[z][y][x]-Field[z][y][x-1])/float32(XStep*(getEx(x)+getEx(x-1))) +
		c.GetLambda(index, index2, x+1, y, x, y)*(Field[z][y][x]-Field[z][y][x+1])/float32(XStep*(getEx(x)+getEx(x+1))) +
		c.GetLambda(index, index3, x, y-1, x, y)*(Field[z][y][x]-Field[z][y-1][x])/float32(YStep*(getEy(y)+getEy(y-1))) +
		c.GetLambda(index, index4, x, y+1, x, y)*(Field[z][y][x]-Field[z][y+1][x])/float32(YStep*(getEy(y)+getEy(y+1)))
	deltaHin = deltaHin * (2 * deltaT / Density[index])
	//fmt.Println(ThermalField[z][y][x]-ThermalField[z][y][x-1], ThermalField[z][y][x]-ThermalField[z][y][x+1], ThermalField[z][y][x]-ThermalField[z][y-1][x], ThermalField[z][y][x]-ThermalField[z][y+1][x], deltaHin/C[index], deltaHin/C[index], "内部点")
	k := z
	if ThermalField.IsFull {
		k = z
		if ThermalField.IsCopy {
			k = (ThermalField.End - ThermalField.Start) - (ThermalField.End - z)
		}
	}
	if c.alternating {
		ThermalField1Ptr[k][y][x] = Field[z][y][x] - deltaHin/C[index] // 需要修改焓的变化到温度变化的映射关系
	} else {
		ThermalFieldPtr[k][y][x] = Field[z][y][x] - deltaHin/C[index]
	}
}

// 对比使用
func (c *Calculator) CalculateSerially(deltaT float32, ThermalField ThermalFieldStruct) {
	var start = time.Now()
	for count := 0; count < 4; count++ {
		for k := 0; k < ZLength/ZStep; k++ {
			// 先计算点，再计算外表面，再计算里面的点
			c.calculatePointLT(deltaT, k, ThermalField)
			for i := 1; i < Length/XStep/2; i++ {
				c.calculatePointTA(deltaT, i, k, ThermalField)
			}
			for j := Width / YStep / 2; j < Width/YStep-1; j++ {
				c.calculatePointLA(deltaT, j, k, ThermalField)
			}
			for j := Width/YStep - 1 - c.EdgeWidth; j < Width/YStep-1; j++ {
				for i := 1; i < 1+c.EdgeWidth; i++ {
					c.calculatePointIN(deltaT, i, j, k, ThermalField)
				}
			}
			for j := Width / YStep / 2; j < Width/YStep-1-c.EdgeWidth; j++ {
				for i := 1; i < 1+c.EdgeWidth; i++ {
					c.calculatePointIN(deltaT, i, j, k, ThermalField)
				}
			}
			for j := Width/YStep - 1 - c.EdgeWidth; j < Width/YStep-1; j++ {
				for i := 1 + c.EdgeWidth; i < Length/XStep/2; i = i + 1 {
					c.calculatePointIN(deltaT, i, j, k, ThermalField)
				}
			}
			for j := Width / YStep / 2; j < Width/YStep-1-c.EdgeWidth; j = j + c.Step {
				for i := 1 + c.EdgeWidth; i < Length/XStep/2; i = i + c.Step {
					c.calculatePointIN(deltaT, i, j, k, ThermalField)
				}
			}
		}
	}
	fmt.Println("串行计算时间: ", time.Since(start))
}

// 并行计算
func (c *Calculator) calculateCase1(deltaT float32, ThermalField ThermalFieldStruct) {
	var start = time.Now()
	var count = 0
	for k := ThermalField.Start; k < ThermalField.End; k++ {
		// 先计算点，再计算外表面，再计算里面的点
		c.calculatePointLT(deltaT, k, ThermalField)
		count++
		for i := 1; i < Length/XStep/2; i++ {
			c.calculatePointTA(deltaT, i, k, ThermalField)
			count++
		}
		for j := Width / YStep / 2; j < Width/YStep-1; j++ {
			c.calculatePointLA(deltaT, j, k, ThermalField)
			count++
		}
		for j := Width/YStep - 1 - c.EdgeWidth; j < Width/YStep-1; j++ {
			for i := 1; i < 1+c.EdgeWidth; i++ {
				c.calculatePointIN(deltaT, i, j, k, ThermalField)
				count++
			}
		}
		for j := Width / YStep / 2; j < Width/YStep-1-c.EdgeWidth; j++ {
			for i := 1; i < 1+c.EdgeWidth; i++ {
				c.calculatePointIN(deltaT, i, j, k, ThermalField)
				count++
			}
		}
		for j := Width/YStep - 1 - c.EdgeWidth; j < Width/YStep-1; j++ {
			for i := 1 + c.EdgeWidth; i < Length/XStep/2; i = i + 1 {
				c.calculatePointIN(deltaT, i, j, k, ThermalField)
				count++
			}
		}
		for j := Width / YStep / 2; j < Width/YStep-1-c.EdgeWidth; j = j + c.Step {
			for i := 1 + c.EdgeWidth; i < Length/XStep/2; i = i + c.Step {
				c.calculatePointIN(deltaT, i, j, k, ThermalField)
				count++
			}
		}
	}

	fmt.Println("任务1执行时间: ", time.Since(start), "总共计算：", count, "个点")
}

func (c *Calculator) calculateCase2(deltaT float32, ThermalField ThermalFieldStruct) {
	var start = time.Now()
	var count = 0
	for k := ThermalField.Start; k < ThermalField.End; k++ {
		// 先计算点，再计算外表面，再计算里面的点
		c.calculatePointRT(deltaT, k, ThermalField)
		count++
		for i := Length / XStep / 2; i < Length/XStep-1; i++ {
			c.calculatePointTA(deltaT, i, k, ThermalField)
			count++
		}
		for j := Width / YStep / 2; j < Width/YStep-1; j++ {
			c.calculatePointRA(deltaT, j, k, ThermalField)
			count++
		}
		for j := Width/YStep - 1 - c.EdgeWidth; j < Width/YStep-1; j++ {
			for i := Length/XStep - 1 - c.EdgeWidth; i < Length/XStep-1; i++ {
				c.calculatePointIN(deltaT, i, j, k, ThermalField)
				count++
			}
		}
		for j := Width / YStep / 2; j < Width/YStep-1-c.EdgeWidth; j++ {
			for i := Length/XStep - 1 - c.EdgeWidth; i < Length/XStep-1; i++ {
				c.calculatePointIN(deltaT, i, j, k, ThermalField)
				count++
			}
		}
		for j := Width/YStep - 1 - c.EdgeWidth; j < Width/YStep-1; j++ {
			for i := Length / XStep / 2; i < Length/XStep-1-c.EdgeWidth; i = i + 1 {
				c.calculatePointIN(deltaT, i, j, k, ThermalField)
				count++
			}
		}
		for j := Width / YStep / 2; j < Width/YStep-1-c.EdgeWidth; j = j + c.Step {
			for i := Length / XStep / 2; i < Length/XStep-1-c.EdgeWidth; i = i + c.Step {
				c.calculatePointIN(deltaT, i, j, k, ThermalField)
				count++
			}
		}
	}
	fmt.Println("任务2执行时间: ", time.Since(start), "总共计算：", count, "个点")
}

func (c *Calculator) calculateCase3(deltaT float32, ThermalField ThermalFieldStruct) {
	var start = time.Now()
	var count = 0
	for k := ThermalField.Start; k < ThermalField.End; k++ {
		// 先计算点，再计算外表面，再计算里面的点
		c.calculatePointRB(deltaT, k, ThermalField)
		count++
		for i := Length / XStep / 2; i < Length/XStep-1; i++ {
			c.calculatePointBA(deltaT, i, k, ThermalField)
			count++
		}
		for j := 1; j < Width/YStep/2; j++ {
			c.calculatePointRA(deltaT, j, k, ThermalField)
			count++
		}
		for j := 1; j < 1+c.EdgeWidth; j++ {
			for i := Length/XStep - 1 - c.EdgeWidth; i < Length/XStep-1; i++ {
				c.calculatePointIN(deltaT, i, j, k, ThermalField)
				count++
			}
		}
		for j := 1 + c.EdgeWidth; j < Width/YStep/2; j++ {
			for i := Length/XStep - 1 - c.EdgeWidth; i < Length/XStep-1; i++ {
				c.calculatePointIN(deltaT, i, j, k, ThermalField)
				count++
			}
		}
		for j := 1; j < 1+c.EdgeWidth; j++ {
			for i := Length / XStep / 2; i < Length/XStep-1-c.EdgeWidth; i++ {
				c.calculatePointIN(deltaT, i, j, k, ThermalField)
				count++
			}
		}
		for j := 1 + c.EdgeWidth; j < Width/YStep/2; j = j + c.Step {
			for i := Length / XStep / 2; i < Length/XStep-1-c.EdgeWidth; i = i + c.Step {
				c.calculatePointIN(deltaT, i, j, k, ThermalField)
				count++
			}
		}
	}
	fmt.Println("任务3执行时间: ", time.Since(start), "总共计算：", count, "个点")
}

func (c *Calculator) calculateCase4(deltaT float32, ThermalField ThermalFieldStruct) {
	var start = time.Now()
	var count = 0
	for k := ThermalField.Start; k < ThermalField.End; k++ {
		// 先计算点，再计算外表面，再计算里面的点
		c.calculatePointLB(deltaT, k, ThermalField)
		count++
		for i := 1; i < Length/XStep/2; i++ {
			c.calculatePointBA(deltaT, i, k, ThermalField)
			count++
		}
		for j := 1; j < Width/YStep/2; j++ {
			c.calculatePointLA(deltaT, j, k, ThermalField)
			count++
		}
		for j := 1; j < 1+c.EdgeWidth; j++ {
			for i := 1; i < 1+c.EdgeWidth; i++ {
				c.calculatePointIN(deltaT, i, j, k, ThermalField)
				count++
			}
		}
		for j := 1 + c.EdgeWidth; j < Width/YStep/2; j++ {
			for i := 1; i < 1+c.EdgeWidth; i++ {
				c.calculatePointIN(deltaT, i, j, k, ThermalField)
				count++
			}
		}
		for j := 1; j < 1+c.EdgeWidth; j++ {
			for i := 1 + c.EdgeWidth; i < Length/XStep/2; i++ {
				c.calculatePointIN(deltaT, i, j, k, ThermalField)
				count++
			}
		}
		for j := 1 + c.EdgeWidth; j < Width/YStep/2; j = j + c.Step {
			for i := 1 + c.EdgeWidth; i < Length/XStep/2; i = i + c.Step {
				c.calculatePointIN(deltaT, i, j, k, ThermalField)
				count++
			}
		}
	}
	fmt.Println("任务4执行时间: ", time.Since(start), "总共计算：", count, "个点")
}

func (c *Calculator) calculate1(deltaT float32) {
	c.calculateCase1(deltaT, ThermalFieldCopyPtr)
	c.calculateCase1(deltaT, c.ThermalField)
}

func (c *Calculator) calculate2(deltaT float32) {
	c.calculateCase2(deltaT, ThermalFieldCopyPtr)
	c.calculateCase2(deltaT, c.ThermalField)
}

func (c *Calculator) calculate3(deltaT float32) {
	c.calculateCase3(deltaT, ThermalFieldCopyPtr)
	c.calculateCase3(deltaT, c.ThermalField)
}

func (c *Calculator) calculate4(deltaT float32) {
	c.calculateCase4(deltaT, ThermalFieldCopyPtr)
	c.calculateCase4(deltaT, c.ThermalField)
}

func (c *Calculator) CalculateConcurrently(deltaT float32) time.Duration {
	var start = time.Now()
	var wg = sync.WaitGroup{}
	wg.Add(4)
	go func() {
		c.calculate1(deltaT)
		wg.Done()
	}()
	go func() {
		c.calculate2(deltaT)
		wg.Done()
	}()
	go func() {
		c.calculate3(deltaT)
		wg.Done()
	}()
	go func() {
		c.calculate4(deltaT)
		wg.Done()
	}()
	wg.Wait()
	fmt.Println("并行计算时间：", time.Since(start))
	return time.Since(start)
}

func (c *Calculator) Run() {
	// 先计算timeStep
	duration := time.Second * 0
	//count := 0
LOOP:
	for {
		//if count > 0 {
		//	return
		//}
		select {
		case <-c.CalcHub.Stop:
			break LOOP
		default:
			deltaT, _ := c.calculateTimeStep()
			calcDuration := c.CalculateConcurrently(deltaT) // c.ThermalField.Field 最开始赋值为 ThermalField对应的指针
			if calcDuration < 250*time.Millisecond {
				calcDuration = 250 * time.Millisecond
			}
			duration += calcDuration
			if c.alternating {
				c.ThermalField.Field = ThermalField1Ptr
			} else {
				c.ThermalField.Field = ThermalFieldPtr
			}
			for i := Width/YStep - 1; i > Width/YStep-6; i-- {
				for j := Length/XStep - 5; j <= Length/XStep-1; j++ {
					fmt.Print(math.Floor(float64(c.ThermalField.Field[c.ThermalField.End-1][i][j])), " ")
				}
				fmt.Print(i)
				fmt.Println()
			}
			// todo 这里需要根据准确的deltaT来确定时间步长
			c.updateSliceInfo(calcDuration)
			c.alternating = !c.alternating // 仅在这里修改
			fmt.Println("计算温度场花费的时间：", duration)
			if duration > time.Second*4 {
				c.CalcHub.PushSignal()
				//count++
				duration = time.Second * 0
			}
		}
	}
}

func (c *Calculator) updateSliceInfo(calcDuration time.Duration) {
	v := c.V // m/min -> mm/s
	var distance int64
	distance = v*calcDuration.Microseconds() + c.reminder
	if distance == 0 {
		return
	}
	c.reminder = distance % 1e7 // Microseconds = 1e6 and zStep = 10
	newSliceNum := distance / 1e7

	if c.ThermalField.IsTail {
		// 处理拉尾坯的阶段
		add := int(newSliceNum)
		c.ThermalField.Start += add
		if c.ThermalField.Start > c.ThermalField.End {
			c.ThermalField.Start = c.ThermalField.End
		}
		// todo 处理不再进入新切片的情况，也需要考虑再次进入新切片时如何重新开始计算
		return
	}

	if c.ThermalField.IsFull {
		add := int(newSliceNum)             // 加入的新切片数
		ThermalFieldCopyPtr.Start -= add    // start 初始 等于 end 等于 最大切片下标
		if ThermalFieldCopyPtr.Start <= 0 { // 新加入的切片组成了一个新的三维数组, 通过交换指针将新生成的三维数组变为温度场计算中指针的指向
			add = -ThermalFieldCopyPtr.Start
			{ // 交换
				tmp := c.ThermalField.Field
				c.ThermalField.Field = ThermalFieldCopyPtr.Field
				ThermalFieldCopyPtr.Field = tmp
			}
			ThermalFieldCopyPtr.End = ZLength / ZStep
			ThermalFieldCopyPtr.Start = ThermalFieldCopyPtr.End - add
			c.ThermalField.IsCopy = false
			ThermalFieldCopyPtr.IsCopy = true
			c.ThermalField.Start = ZLength / ZStep
			c.ThermalField.End = ZLength / ZStep
		}

		c.ThermalField.End -= add
		// 新加入的切片未组成一个三维数组
		for z := ThermalFieldCopyPtr.Start; z < ThermalFieldCopyPtr.Start+add; z++ {
			for y := 0; y < Width/YStep; y++ {
				for x := 0; z < Length/XStep; z++ {
					ThermalFieldCopyPtr.Field[z][y][x] = c.initialTemperature
				}
			}
		}
	} else {
		c.ThermalField.Start -= int(newSliceNum)
		if c.ThermalField.Start <= 0 {
			c.ThermalField.Start = 0
			c.ThermalField.IsFull = true
			ThermalFieldCopyPtr.IsFull = true
			return
		}
		fmt.Println("目前的切片数为：", c.ThermalField.End-c.ThermalField.Start)
	}
}

func (c *Calculator) test() {
	start := time.Now()
	var newSlice [Width / YStep][Length / XStep]float32
	for j := 0; j < Width/YStep; j++ {
		for i := 0; i < Length/XStep; i++ {
			newSlice[j][i] = 1550
		}
	}
	fmt.Println(time.Since(start))
	for z := 0; z < ZLength/ZStep-1; z++ {
		ThermalField[z+1] = ThermalField1[z]
	}

	ThermalField[0] = newSlice
	fmt.Println(time.Since(start))
}

func (c *Calculator) Calculate() {
	c.ThermalField.Start = 0
	c.ThermalField.End = 4000
	// 四个核心一起计算
	//for count := 0; count < 100; count++ {
	//	deltaT, _ := c.calculateTimeStep()
	//	c.CalculateConcurrently(deltaT)
	//	for i := 0; i < 100; i++ {
	//		for j := 0; j < len(ThermalField[i]); j++ {
	//			for k := 0; k < len(ThermalField[i][j]); k++ {
	//				if c.ThermalField[i][j][k] > 1550 || c.ThermalField1[i][j][k] > 1550 {
	//					fmt.Print(c.ThermalField[i][j][k], c.ThermalField1[i][j][k])
	//				}
	//			}
	//		}
	//	}
	//}

	for count := 0; count < 10; count++ {
		//c.calculatePointLB(deltaT, 0)
		//c.calculatePointBA(deltaT, 1, 0)
		//c.calculatePointBA(deltaT, Length/XStep-2, 0)
		//
		//c.calculatePointRB(deltaT, 0)
		//c.calculatePointRA(deltaT, 1, 0)
		//c.calculatePointRA(deltaT, Width/YStep-2, 0)
		//
		//c.calculatePointRT(deltaT, 0)
		//c.calculatePointTA(deltaT, 1, 0)
		//c.calculatePointTA(deltaT, Length/XStep-2, 0)
		//
		//c.calculatePointLT(deltaT, 0)
		//c.calculatePointLA(deltaT, 1, 0)
		//c.calculatePointLA(deltaT, Width/YStep-2, 0)
		//
		//c.calculatePointIN(deltaT, Length/YStep-2, Width/YStep-2, 0)
		//fmt.Println("----------------")
		//fmt.Println("左下角：", c.ThermalField[0][0][0], c.ThermalField1[0][0][0])
		//fmt.Println("下面边：", c.ThermalField[0][0][1], c.ThermalField1[0][0][1])
		//fmt.Println("下面边：", c.ThermalField[0][0][Length/XStep-2], c.ThermalField1[0][0][Length/XStep-2])
		//
		//fmt.Println("右下角：", c.ThermalField[0][0][Length/XStep-1], c.ThermalField1[0][0][Length/XStep-1])
		//fmt.Println("右面边：", c.ThermalField[0][1][Length/XStep-1], c.ThermalField1[0][1][Length/XStep-1])
		//fmt.Println("右面边：", c.ThermalField[0][Width/YStep-2][Length/XStep-1], c.ThermalField1[0][Width/YStep-2][Length/XStep-1])
		//
		//fmt.Println("右上角：", c.ThermalField[0][Width/YStep-1][Length/XStep-1], c.ThermalField1[0][Width/YStep-1][Length/XStep-1])
		//fmt.Println("上面边：", c.ThermalField[0][Width/YStep-1][1], c.ThermalField1[0][Width/YStep-1][1])
		//fmt.Println("上面边：", c.ThermalField[0][Width/YStep-1][Length/XStep-2], c.ThermalField1[0][Width/YStep-1][Length/XStep-2])
		//
		//fmt.Println("左上角：", c.ThermalField[0][Width/YStep-1][0], c.ThermalField1[0][Width/YStep-1][0])
		//fmt.Println("左面边：", c.ThermalField[0][1][0], c.ThermalField1[0][1][0])
		//fmt.Println("左面边：", c.ThermalField[0][Width/YStep-2][0], c.ThermalField1[0][Width/YStep-2][0])
		//
		//fmt.Println("内部点：", c.ThermalField[0][Width/YStep-2][Length/XStep-2], c.ThermalField1[0][Width/YStep-2][Length/XStep-2])
		//fmt.Println()
		fmt.Println(c.alternating)
		deltaT, _ := c.calculateTimeStep()

		c.CalculateConcurrently(deltaT)
		if c.alternating {
			c.ThermalField.Field = ThermalField1Ptr
		} else {
			c.ThermalField.Field = ThermalFieldPtr
		}
		fmt.Println(c.ThermalField.Field[0][Width/YStep-1][Length/XStep-1])

		for i := Width/YStep - 1; i > Width/YStep-6; i-- {
			for j := Length/XStep - 5; j <= Length/XStep-1; j++ {
				fmt.Print(math.Floor(float64(c.ThermalField.Field[0][i][j])), " ")
			}
			fmt.Print(i)
			fmt.Println()
		}
		c.alternating = !c.alternating
	}
	// 一个核心计算
	//c.CalculateSerially()
}

//func (c *Calculator) GetDeltaT(z, x, y int) float32 {
//	var c.ThermalField.Field *[ZLength / ZStep][Width / YStep][Length / XStep]float32
//	if c.alternating {
//		ThermalFieldCopy = ThermalField1Ptr
//	} else {
//		ThermalFieldCopy = ThermalFieldPtr
//	}
//	var t = ThermalFieldCopy[z][y][x]
//	var index = int(t)/5 - 1
//	var index1, index2, index3, index4 int
//	var denominator = float32(1.0)
//	if x == 0 && y == 0 { // case 1
//		index1 = int(ThermalFieldCopy[z][y][x+1])/5 - 1
//		index2 = int(ThermalFieldCopy[z][y+1][x])/5 - 1
//		denominator = 2*c.GetLambda(index, index1, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
//			2*c.GetLambda(index, index2, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1)))
//	} else if x > 0 && x < Length/XStep-1 && y == 0 { // case 2
//		index1 = int(ThermalFieldCopy[z][y][x-1])/5 - 1
//		index2 = int(ThermalFieldCopy[z][y][x+1])/5 - 1
//		index3 = int(ThermalFieldCopy[z][y+1][x])/5 - 1
//		denominator = 2*c.GetLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
//			2*c.GetLambda(index, index2, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
//			2*c.GetLambda(index, index3, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1)))
//	} else if x == Length/XStep-1 && y == 0 { // case 3
//		index1 = int(ThermalFieldCopy[z][y][x-1])/5 - 1
//		index2 = int(ThermalFieldCopy[z][y+1][x])/5 - 1
//		denominator = 2*c.GetLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
//			2*c.GetLambda(index, index2, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1))) +
//			c.HEff[index]/(XStep)
//	} else if x == 0 && y > 0 && y < Width/YStep-1 { // case 4
//		index1 = int(ThermalFieldCopy[z][y][x+1])/5 - 1
//		index2 = int(ThermalFieldCopy[z][y+1][x])/5 - 1
//		index3 = int(ThermalFieldCopy[z][y-1][x])/5 - 1
//		denominator = 2*c.GetLambda(index, index1, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
//			2*c.GetLambda(index, index2, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1))) +
//			2*c.GetLambda(index, index3, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1)))
//	} else if x > 0 && x < Length/XStep-1 && y > 0 && y < Width/YStep-1 { // case 5
//		index1 = int(ThermalFieldCopy[z][y][x-1])/5 - 1
//		index2 = int(ThermalFieldCopy[z][y][x+1])/5 - 1
//		index3 = int(ThermalFieldCopy[z][y+1][x])/5 - 1
//		index4 = int(ThermalFieldCopy[z][y-1][x])/5 - 1
//		denominator = 2*c.GetLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
//			2*c.GetLambda(index, index2, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
//			2*c.GetLambda(index, index3, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1))) +
//			2*c.GetLambda(index, index4, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1)))
//	} else if x == Length/XStep-1 && y > 0 && y < Width/YStep-1 { // case6
//		index1 = int(ThermalFieldCopy[z][y][x-1])/5 - 1
//		index2 = int(ThermalFieldCopy[z][y+1][x])/5 - 1
//		index3 = int(ThermalFieldCopy[z][y-1][x])/5 - 1
//		denominator = 2*c.GetLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
//			2*c.GetLambda(index, index2, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1))) +
//			2*c.GetLambda(index, index3, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1))) +
//			c.HEff[index]/(XStep)
//	} else if x == 0 && y == Width/YStep-1 { // case7
//		index1 = int(ThermalFieldCopy[z][y][x+1])/5 - 1
//		index2 = int(ThermalFieldCopy[z][y-1][x])/5 - 1
//		denominator = 2*c.GetLambda(index, index1, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
//			2*c.GetLambda(index, index2, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1))) +
//			c.HEff[index]/(YStep)
//	} else if x > 0 && x < Length/XStep-1 && y == Width/YStep-1 { // case 8
//		index1 = int(ThermalFieldCopy[z][y][x-1])/5 - 1
//		index2 = int(ThermalFieldCopy[z][y][x+1])/5 - 1
//		index3 = int(ThermalFieldCopy[z][y-1][x])/5 - 1
//		denominator = 2*c.GetLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
//			2*c.GetLambda(index, index2, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
//			2*c.GetLambda(index, index3, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1))) +
//			c.HEff[index]/(YStep)
//	} else if x == Length/XStep-1 && y == Width/YStep-1 { // case 9
//		index1 = int(ThermalFieldCopy[z][y][x-1])/5 - 1
//		index2 = int(ThermalFieldCopy[z][y-1][x])/5 - 1
//		denominator = 2*c.GetLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
//			2*c.GetLambda(index, index2, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1))) +
//			c.HEff[index]/(XStep) +
//			c.HEff[index]/(YStep)
//	}
//
//	return (c.Density[index] * c.Enthalpy[index]) / (t * denominator)
//}
