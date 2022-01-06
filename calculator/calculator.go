package calculator

import (
	"fmt"
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
	ArrayLength = 1600 / 5
)

type Calculator struct {
	// 计算参数
	EdgeWidth int

	Step     int                  // 当c.EdgeWidth > 0, step = 2;
	Density  [ArrayLength]float32 // 密度
	Enthalpy [ArrayLength]float32 // 焓
	Lambda   [ArrayLength]float32 // 导热系数
	HEff     [ArrayLength]float32 // 综合换热系数, 注意：简单处理了！
	Q        [ArrayLength]float32 // 热流密度, 注意：简单处理了
	C        [ArrayLength]float32 // 比热容

	ThermalField  [ZLength / ZStep][Width / YStep][Length / XStep]float32
	ThermalField1 [ZLength / ZStep][Width / YStep][Length / XStep]float32

	// 每计算一个 ▲t 进行一次异或运算
	alternating bool

	//v int 拉速
	v int
}

func NewCalculator(edgeWidth int) Calculator {
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
	for z := 0; z < ZLength/ZStep; z++ {
		for y := 0; y < Width/YStep; y++ {
			for x := 0; x < Length/XStep; x++ {
				calculator.ThermalField[z][y][x] = 1600.0
				calculator.ThermalField1[z][y][x] = 1600.0
			}
		}
	}

	// 2. 导热系数，200℃ 到 1600℃，随温度的上升而下降
	var LambdaStart = float32(50.0)
	var LambdaIter = float32(50.0-45.0) / 1600 / 5
	for i := 0; i < ArrayLength; i++ {
		calculator.Lambda[i] = LambdaStart - float32(i)*LambdaIter
	}

	// 3. 密度
	var DensityStart = float32(8.0)
	var DensityIter = float32(8.0-7.0) / 1600 / 5
	for i := 0; i < ArrayLength; i++ {
		calculator.Density[i] = DensityStart - float32(i)*DensityIter
	}

	// 4. 焓值
	var EnthalpyStart = float32(1000.0)
	var EnthalpyStep = float32(10000.0-1000.0) / 1600 / 5
	for i := 0; i < ArrayLength; i++ {
		calculator.Enthalpy[i] = EnthalpyStart + float32(i)*EnthalpyStep
	}

	// 5. 综合换热系数
	var HEffStart = float32(5.0)
	var HEffStep = float32(20.0-15.0) / 1600 / 5
	for i := 0; i < ArrayLength; i++ {
		calculator.HEff[i] = HEffStart + float32(i)*HEffStep
	}

	// 6. 热流密度
	var QStart = float32(12.0)
	var QStep = float32(25.0-20.0) / 1600 / 5
	for i := 0; i < ArrayLength; i++ {
		calculator.Q[i] = QStart + float32(i)*QStep
	}

	// 7. 比热容
	var CStart = float32(46.0)
	var CStep = float32(754.0) / 1600 / 5
	for i := 0; i < ArrayLength; i++ {
		calculator.C[i] = CStart + float32(i)*CStep
	}

	calculator.EdgeWidth = edgeWidth
	calculator.Step = 1
	fmt.Println(calculator.EdgeWidth)
	if calculator.EdgeWidth > 0 {
		calculator.Step = 2
	}

	fmt.Println("初始化时间: ", time.Since(start))
	return calculator
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
		return K * c.Lambda[index1] * c.Lambda[index2] * float32(ex1+ex2) /
			(c.Lambda[index1]*float32(ex2) + c.Lambda[index2]*float32(ex1))
	}
	if y1 != y2 {
		return K * c.Lambda[index1] * c.Lambda[index2] * float32(ey1+ey2) /
			(c.Lambda[index1]*float32(ey2) + c.Lambda[index2]*float32(ey1))
	}
	return 1.0 // input error
}

// 计算时间步长
func (c *Calculator) GetDeltaT(z, x, y int) float32 {
	var t = c.ThermalField[z][y][x]
	var index = int(t)/5 - 1
	var index1, index2, index3, index4 int
	var denominator = float32(1.0)
	if x == 0 && y == 0 { // case 1
		index1 = int(c.ThermalField[z][y][x+1])/5 - 1
		index2 = int(c.ThermalField[z][y+1][x])/5 - 1
		denominator = 2*c.GetLambda(index, index1, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
			2*c.GetLambda(index, index2, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1)))
	} else if x > 0 && x < Length/XStep-1 && y == 0 { // case 2
		index1 = int(c.ThermalField[z][y][x-1])/5 - 1
		index2 = int(c.ThermalField[z][y][x+1])/5 - 1
		index3 = int(c.ThermalField[z][y+1][x])/5 - 1
		denominator = 2*c.GetLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
			2*c.GetLambda(index, index2, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
			2*c.GetLambda(index, index3, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1)))
	} else if x == Length/XStep-1 && y == 0 { // case 3
		index1 = int(c.ThermalField[z][y][x-1])/5 - 1
		index2 = int(c.ThermalField[z][y+1][x])/5 - 1
		denominator = 2*c.GetLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
			2*c.GetLambda(index, index2, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1))) +
			c.HEff[index]/(XStep)
	} else if x == 0 && y > 0 && y < Width/YStep-1 { // case 4
		index1 = int(c.ThermalField[z][y][x+1])/5 - 1
		index2 = int(c.ThermalField[z][y+1][x])/5 - 1
		index3 = int(c.ThermalField[z][y-1][x])/5 - 1
		denominator = 2*c.GetLambda(index, index1, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
			2*c.GetLambda(index, index2, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1))) +
			2*c.GetLambda(index, index3, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1)))
	} else if x > 0 && x < Length/XStep-1 && y > 0 && y < Width/YStep-1 { // case 5
		index1 = int(c.ThermalField[z][y][x-1])/5 - 1
		index2 = int(c.ThermalField[z][y][x+1])/5 - 1
		index3 = int(c.ThermalField[z][y+1][x])/5 - 1
		index4 = int(c.ThermalField[z][y-1][x])/5 - 1
		denominator = 2*c.GetLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
			2*c.GetLambda(index, index2, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
			2*c.GetLambda(index, index3, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1))) +
			2*c.GetLambda(index, index4, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1)))
	} else if x == Length/XStep-1 && y > 0 && y < Width/YStep-1 { // case6
		index1 = int(c.ThermalField[z][y][x-1])/5 - 1
		index2 = int(c.ThermalField[z][y+1][x])/5 - 1
		index3 = int(c.ThermalField[z][y-1][x])/5 - 1
		denominator = 2*c.GetLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
			2*c.GetLambda(index, index2, x, y, x, y+1)/float32(YStep*(getEy(y)+getEy(y+1))) +
			2*c.GetLambda(index, index3, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1))) +
			c.HEff[index]/(XStep)
	} else if x == 0 && y == Width/YStep-1 { // case7
		index1 = int(c.ThermalField[z][y][x+1])/5 - 1
		index2 = int(c.ThermalField[z][y-1][x])/5 - 1
		denominator = 2*c.GetLambda(index, index1, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
			2*c.GetLambda(index, index2, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1))) +
			c.HEff[index]/(YStep)
	} else if x > 0 && x < Length/XStep-1 && y == Width/YStep-1 { // case 8
		index1 = int(c.ThermalField[z][y][x-1])/5 - 1
		index2 = int(c.ThermalField[z][y][x+1])/5 - 1
		index3 = int(c.ThermalField[z][y-1][x])/5 - 1
		denominator = 2*c.GetLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
			2*c.GetLambda(index, index2, x, y, x+1, y)/float32(XStep*(getEx(x)+getEx(x+1))) +
			2*c.GetLambda(index, index3, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1))) +
			c.HEff[index]/(YStep)
	} else if x == Length/XStep-1 && y == Width/YStep-1 { // case 9
		index1 = int(c.ThermalField[z][y][x-1])/5 - 1
		index2 = int(c.ThermalField[z][y-1][x])/5 - 1
		denominator = 2*c.GetLambda(index, index1, x, y, x-1, y)/float32(XStep*(getEx(x)+getEx(x-1))) +
			2*c.GetLambda(index, index2, x, y, x, y-1)/float32(YStep*(getEy(y)+getEy(y-1))) +
			c.HEff[index]/(XStep) +
			c.HEff[index]/(YStep)
	}

	return (c.Density[index] * c.Enthalpy[index]) / (t * denominator)
}

// 计算时间步长
func (c *Calculator) calculateTimeStep() float32 {
	// 计算时间步长 - start
	var deltaTArr = [5]float32{}
	deltaTArr[0] = c.GetDeltaT(0, 0, Width/YStep-1)
	deltaTArr[1] = c.GetDeltaT(0, 1, Width/YStep-1)
	deltaTArr[2] = c.GetDeltaT(0, Length/XStep-1, Width/YStep-1)
	deltaTArr[3] = c.GetDeltaT(0, Length/XStep-1, 1)
	deltaTArr[4] = c.GetDeltaT(0, Length/XStep-1, 0)
	var min = float32(1000.0) // 模拟一个很大的数
	for _, i := range deltaTArr {
		if min > i {
			min = i
		}
	}
	return min
	// 计算时间步长 - end
}

// 计算一个left top点的温度变化
func (c *Calculator) calculatePointLT(deltaT float32, z int) {
	var index = int(c.ThermalField[z][Width/YStep-1][0])/5 - 1
	var index1 = int(c.ThermalField[z][Width/YStep-1][1])/5 - 1
	var index2 = int(c.ThermalField[z][Width/YStep-2][0])/5 - 1
	var deltaHlt = c.GetLambda(index, index1, 0, Width/YStep-1, 1, Width/YStep-1)*
		float32(int(c.ThermalField[z][Width/YStep-1][1]-c.ThermalField[z][Width/YStep-1][0]))/
		float32(XStep*(getEx(1)+getEx(0))) +
		c.GetLambda(index, index2, 0, Width/YStep-1, 0, Width/YStep-2)*
			float32(int(c.ThermalField[z][Width/YStep-2][0]-c.ThermalField[z][Width/YStep-1][0]))/
			float32(YStep*(getEy(Width/YStep-1)+getEx(Width/YStep-1))) +
		c.Q[index]/(2*YStep)

	deltaHlt = deltaHlt * (2 * deltaT / c.Density[index])

	if c.alternating {
		c.ThermalField1[z][Width/YStep-1][0] = c.ThermalField[z][Width/YStep-1][0] - deltaHlt/c.C[index]
	} else {
		// 需要修改焓的变化到温度变化的映射关系
		c.ThermalField[z][Width/YStep-1][0] = c.ThermalField1[z][Width/YStep-1][0] - (deltaHlt / c.C[index])
	}
}

// 计算上表面点温度变化
func (c *Calculator) calculatePointTA(deltaT float32, x, z int) {
	var index = int(c.ThermalField[z][Width/YStep-1][x])/5 - 1
	var index1 = int(c.ThermalField[z][Width/YStep-1][x-1])/5 - 1
	var index2 = int(c.ThermalField[z][Width/YStep-1][x+1])/5 - 1
	var index3 = int(c.ThermalField[z][Width/YStep-2][x])/5 - 1
	var deltaHta = c.GetLambda(index, index1, x, Width/YStep-1, x-1, Width/YStep-1)*
		float32(int(c.ThermalField[z][Width/YStep-1][x-1]-c.ThermalField[z][Width/YStep-1][x]))/
		float32(XStep*(getEx(x-1)+getEx(x))) +
		c.GetLambda(index, index2, x, Width/YStep-1, x+1, Width/YStep-1)*
			float32(int(c.ThermalField[z][Width/YStep-1][x+1]-c.ThermalField[z][Width/YStep-1][x]))/
			float32(XStep*(getEx(x)+getEx(x+1))) +
		c.GetLambda(index, index3, x, Width/YStep-1, x, Width/YStep-2)*
			float32(int(c.ThermalField[z][Width/YStep-2][x]-c.ThermalField[z][Width/YStep-1][x]))/
			float32(YStep*(getEy(Width/YStep-2)+getEy(Width/YStep-1))) +
		c.Q[index]/(2*YStep)

	deltaHta = deltaHta * (2 * deltaT / c.Density[index])

	if c.alternating {
		c.ThermalField1[z][Width/YStep-1][x] = c.ThermalField[z][Width/YStep-1][x] - deltaHta/c.C[index]
	} else {
		// 需要修改焓的变化到温度变化的映射关系
		c.ThermalField[z][Width/YStep-1][x] = c.ThermalField1[z][Width/YStep-1][x] - deltaHta/c.C[index]
	}
}

// 计算right top点的温度变化
func (c *Calculator) calculatePointRT(deltaT float32, z int) {
	var index = int(c.ThermalField[z][Width/YStep-1][Length/XStep-1])/5 - 1
	var index1 = int(c.ThermalField[z][Width/YStep-1][Length/XStep-2])/5 - 1
	var index2 = int(c.ThermalField[z][Width/YStep-2][Length/XStep-1])/5 - 1
	var deltaHrt = c.GetLambda(index, index1, Length/XStep-1, Width/YStep-1, Length/XStep-2, Width/YStep-1)*
		float32(int(c.ThermalField[z][Width/YStep-1][Length/XStep-2]-c.ThermalField[z][Width/YStep-1][Length/XStep-1]))/
		float32(XStep*(getEx(Length/XStep-2)+getEx(Length/XStep-1))) +
		c.GetLambda(index, index2, Length/XStep-1, Width/YStep-1, Length/XStep-1, Width/YStep-2)*
			float32(int(c.ThermalField[z][Width/YStep-2][Length/XStep-1]-c.ThermalField[z][Width/YStep-1][Length/XStep-1]))/
			float32(YStep*(getEy(Width/YStep-2)+getEy(Width/YStep-1))) +
		c.Q[index]/(2*YStep) +
		c.Q[index]/(2*XStep)

	deltaHrt = deltaHrt * (2 * deltaT / c.Density[index])

	if c.alternating {
		c.ThermalField1[z][Width/YStep-1][Width/YStep-1] =
			c.ThermalField[z][Width/YStep-1][Width/YStep-1] - deltaHrt/c.C[index] // 需要修改焓的变化到温度变化的映射关系
	} else {
		c.ThermalField[z][Width/YStep-1][Width/YStep-1] =
			c.ThermalField1[z][Width/YStep-1][Width/YStep-1] - deltaHrt/c.C[index]
	}
}

// 计算右表面点的温度变化
func (c *Calculator) calculatePointRA(deltaT float32, y, z int) {
	var index = int(c.ThermalField[z][y][Length/XStep-1])/5 - 1
	var index1 = int(c.ThermalField[z][y][Length/XStep-2])/5 - 1
	var index2 = int(c.ThermalField[z][y-1][Length/XStep-1])/5 - 1
	var index3 = int(c.ThermalField[z][y+1][Length/XStep-1])/5 - 1
	var deltaHra = c.GetLambda(index, index1, Length/XStep-1, y, Length/XStep-2, y)*
		float32(int(c.ThermalField[z][y][Length/XStep-2]-c.ThermalField[z][y][Length/XStep-1]))/
		float32(XStep*(getEx(Length/XStep-2)+getEx(Length/XStep-1))) +
		c.GetLambda(index, index2, Length/XStep-1, y, Length/XStep-1, y-1)*
			float32(int(c.ThermalField[z][y-1][Length/XStep-1]-c.ThermalField[z][y][Length/XStep-1]))/
			float32(YStep*(getEy(y-1)+getEy(y))) +
		c.GetLambda(index, index3, Length/XStep-1, y, Length/XStep-1, y+1)*
			float32(int(c.ThermalField[z][y+1][Length/XStep-1]-c.ThermalField[z][y][Length/XStep-1]))/
			float32(YStep*(getEy(y+1)+getEy(y))) +
		c.Q[index]/(2*XStep)

	deltaHra = deltaHra * (2 * deltaT / c.Density[index])

	if c.alternating {
		c.ThermalField1[z][y][Width/YStep-1] = c.ThermalField[z][y][Width/YStep-1] - deltaHra/c.C[index]
		// 需要修改焓的变化到温度变化的映射关系
	} else {
		c.ThermalField[z][y][Width/YStep-1] = c.ThermalField1[z][y][Width/YStep-1] - deltaHra/c.C[index]
	}
}

// 计算right bottom点的温度变化
func (c *Calculator) calculatePointRB(deltaT float32, z int) {
	var index = int(c.ThermalField[z][0][Length/XStep-1])/5 - 1
	var index1 = int(c.ThermalField[z][0][Length/XStep-2])/5 - 1
	var index2 = int(c.ThermalField[z][1][Length/XStep-1])/5 - 1
	var deltaHrb = c.GetLambda(index, index1, Length/XStep-1, 0, Length/XStep-2, 0)*
		float32(int(c.ThermalField[z][0][Length/XStep-2]-c.ThermalField[z][0][Length/XStep-1]))/
		float32(XStep*(getEx(Length/XStep-2)+getEx(Length/XStep-1))) +
		c.GetLambda(index, index2, Length/XStep-1, 0, Length/XStep-1, 1)*
			float32(int(c.ThermalField[z][1][Length/XStep-1]-c.ThermalField[z][0][Length/XStep-1]))/
			float32(YStep*(getEy(1)+getEy(0))) +
		c.Q[index]/(2*XStep)

	deltaHrb = deltaHrb * (2 * deltaT / c.Density[index])

	if c.alternating {
		c.ThermalField1[z][0][Width/YStep-1] = c.ThermalField[z][0][Width/YStep-1] - deltaHrb/c.C[index] // 需要修改焓的变化到温度变化的映射关系
	} else {
		c.ThermalField[z][0][Width/YStep-1] = c.ThermalField1[z][0][Width/YStep-1] - deltaHrb/c.C[index]
	}
}

// 计算下表面点的温度变化
func (c *Calculator) calculatePointBA(deltaT float32, x, z int) {
	var index = int(c.ThermalField[z][0][x])/5 - 1
	var index1 = int(c.ThermalField[z][0][x-1])/5 - 1
	var index2 = int(c.ThermalField[z][0][x+1])/5 - 1
	var index3 = int(c.ThermalField[z][1][x])/5 - 1
	var deltaHba = c.GetLambda(index, index1, x, 0, x-1, 0)*
		float32(int(c.ThermalField[z][0][x-1]-c.ThermalField[z][0][x]))/
		float32(XStep*(getEx(x-1)+getEx(x))) +
		c.GetLambda(index, index2, x, 0, x+1, 0)*
			float32(int(c.ThermalField[z][0][x+1]-c.ThermalField[z][0][x]))/
			float32(XStep*(getEx(x+1)+getEx(x))) +
		c.GetLambda(index, index3, x, 0, x, 1)*
			float32(int(c.ThermalField[z][1][x]-c.ThermalField[z][0][x]))/
			float32(YStep*(getEy(1)+getEy(0)))

	deltaHba = deltaHba * (2 * deltaT / c.Density[index])

	if c.alternating {
		c.ThermalField1[z][0][x] = c.ThermalField[z][0][x] - deltaHba/c.C[index] // 需要修改焓的变化到温度变化的映射关系
	} else {
		c.ThermalField[z][0][x] = c.ThermalField1[z][0][x] - deltaHba/c.C[index]
	}
}

// 计算left bottom点的温度变化
func (c *Calculator) calculatePointLB(deltaT float32, z int) {
	var index = int(c.ThermalField[z][0][0])/5 - 1
	var index1 = int(c.ThermalField[z][0][1])/5 - 1
	var index2 = int(c.ThermalField[z][1][0])/5 - 1
	var deltaHlb = c.GetLambda(index, index1, 1, 0, 0, 0)*
		float32(int(c.ThermalField[z][0][1]-c.ThermalField[z][0][0]))/
		float32(XStep*(getEx(0)+getEx(1))) +
		c.GetLambda(index, index2, 0, 1, 0, 0)*
			float32(int(c.ThermalField[z][1][0]-c.ThermalField[z][0][0]))/
			float32(YStep*(getEy(1)+getEy(0)))

	deltaHlb = deltaHlb * (2 * deltaT / c.Density[index])

	if c.alternating {
		c.ThermalField1[z][0][0] = c.ThermalField[z][0][0] - deltaHlb/c.C[index] // 需要修改焓的变化到温度变化的映射关系
	} else {
		c.ThermalField[z][0][0] = c.ThermalField1[z][0][0] - deltaHlb/c.C[index]
	}
}

// 计算左表面点温度的变化
func (c *Calculator) calculatePointLA(deltaT float32, y, z int) {
	var index = int(c.ThermalField[z][y][0])/5 - 1
	var index1 = int(c.ThermalField[z][y][1])/5 - 1
	var index2 = int(c.ThermalField[z][y-1][0])/5 - 1
	var index3 = int(c.ThermalField[z][y+1][0])/5 - 1
	var deltaHla = c.GetLambda(index, index1, 1, y, 0, y)*
		float32(int(c.ThermalField[z][y][1]-c.ThermalField[z][y][0]))/
		float32(XStep*(getEx(0)+getEx(1))) +
		c.GetLambda(index, index2, 0, y-1, 0, y)*
			float32(int(c.ThermalField[z][y-1][0]-c.ThermalField[z][y][0]))/
			float32(YStep*(getEy(y)+getEy(y-1))) +
		c.GetLambda(index, index3, 0, y+1, 0, y)*
			float32(int(c.ThermalField[z][y+1][0]-c.ThermalField[z][y][0]))/
			float32(YStep*(getEy(y)+getEy(y+1)))

	deltaHla = deltaHla * (2 * deltaT / c.Density[index])

	if c.alternating {
		c.ThermalField1[z][y][0] = c.ThermalField[z][y][0] - deltaHla/c.C[index] // 需要修改焓的变化到温度变化的映射关系
	} else {
		c.ThermalField[z][y][0] = c.ThermalField1[z][y][0] - deltaHla/c.C[index]
	}
}

// 计算内部点的温度变化
func (c *Calculator) calculatePointIN(deltaT float32, x, y, z int) {
	var index = int(c.ThermalField[z][y][x])/5 - 1
	var index1 = int(c.ThermalField[z][y][x-1])/5 - 1
	var index2 = int(c.ThermalField[z][y][x+1])/5 - 1
	var index3 = int(c.ThermalField[z][y-1][x])/5 - 1
	var index4 = int(c.ThermalField[z][y+1][x])/5 - 1
	var deltaHin = c.GetLambda(index, index1, x-1, y, x, y)*
		float32(int(c.ThermalField[z][y][x-1]-c.ThermalField[z][y][x]))/
		float32(XStep*(getEx(x)+getEx(x-1))) +
		c.GetLambda(index, index2, x+1, y, x, y)*
			float32(int(c.ThermalField[z][y][x+1]-c.ThermalField[z][y][x]))/
			float32(XStep*(getEx(x)+getEx(x+1))) +
		c.GetLambda(index, index3, x, y-1, x, y)*
			float32(int(c.ThermalField[z][y-1][x]-c.ThermalField[z][y][x]))/
			float32(YStep*(getEy(y)+getEy(y-1))) +
		c.GetLambda(index, index4, x, y+1, x, y)*
			float32(int(c.ThermalField[z][y+1][x]-c.ThermalField[z][y][x]))/
			float32(YStep*(getEy(y)+getEy(y+1)))

	deltaHin = deltaHin * (2 * deltaT / c.Density[index])

	if c.alternating {
		c.ThermalField1[z][y][x] = c.ThermalField[z][y][x] - deltaHin/c.C[index] // 需要修改焓的变化到温度变化的映射关系
	} else {
		c.ThermalField[z][y][x] = c.ThermalField1[z][y][x] - deltaHin/c.C[index]
	}
}

func (c *Calculator) CalculateSerially() {
	var start = time.Now()
	for count := 0; count < 4; count++ {
		var deltaT = c.calculateTimeStep()
		for k := 0; k < ZLength/ZStep; k++ {
			// 先计算点，再计算外表面，再计算里面的点
			c.calculatePointLT(deltaT, k)
			for i := 1; i < Length/XStep/2; i++ {
				c.calculatePointTA(deltaT, i, k)
			}
			for j := Width / YStep / 2; j < Width/YStep-1; j++ {
				c.calculatePointLA(deltaT, j, k)
			}
			for j := Width/YStep - 1 - c.EdgeWidth; j < Width/YStep-1; j++ {
				for i := 1; i < 1+c.EdgeWidth; i++ {
					c.calculatePointIN(deltaT, i, j, k)
				}
			}
			for j := Width / YStep / 2; j < Width/YStep-1-c.EdgeWidth; j++ {
				for i := 1; i < 1+c.EdgeWidth; i++ {
					c.calculatePointIN(deltaT, i, j, k)
				}
			}
			for j := Width/YStep - 1 - c.EdgeWidth; j < Width/YStep-1; j++ {
				for i := 1 + c.EdgeWidth; i < Length/XStep/2; i = i + 1 {
					c.calculatePointIN(deltaT, i, j, k)
				}
			}
			for j := Width / YStep / 2; j < Width/YStep-1-c.EdgeWidth; j = j + c.Step {
				for i := 1 + c.EdgeWidth; i < Length/XStep/2; i = i + c.Step {
					c.calculatePointIN(deltaT, i, j, k)
				}
			}
		}
	}
	fmt.Println("串行计算时间: ", time.Since(start))
}

func (c *Calculator) calculateCase1() {
	var start = time.Now()
	var deltaT = c.calculateTimeStep()
	var count = 0
	for k := 0; k < ZLength/ZStep; k++ {
		// 先计算点，再计算外表面，再计算里面的点
		c.calculatePointLT(deltaT, k)
		count++
		for i := 1; i < Length/XStep/2; i++ {
			c.calculatePointTA(deltaT, i, k)
			count++
		}
		for j := Width / YStep / 2; j < Width/YStep-1; j++ {
			c.calculatePointLA(deltaT, j, k)
			count++
		}
		for j := Width/YStep - 1 - c.EdgeWidth; j < Width/YStep-1; j++ {
			for i := 1; i < 1+c.EdgeWidth; i++ {
				c.calculatePointIN(deltaT, i, j, k)
				count++
			}
		}
		for j := Width / YStep / 2; j < Width/YStep-1-c.EdgeWidth; j++ {
			for i := 1; i < 1+c.EdgeWidth; i++ {
				c.calculatePointIN(deltaT, i, j, k)
				count++
			}
		}
		for j := Width/YStep - 1 - c.EdgeWidth; j < Width/YStep-1; j++ {
			for i := 1 + c.EdgeWidth; i < Length/XStep/2; i = i + 1 {
				c.calculatePointIN(deltaT, i, j, k)
				count++
			}
		}
		for j := Width / YStep / 2; j < Width/YStep-1-c.EdgeWidth; j = j + c.Step {
			for i := 1 + c.EdgeWidth; i < Length/XStep/2; i = i + c.Step {
				c.calculatePointIN(deltaT, i, j, k)
				count++
			}
		}
		c.alternating = !c.alternating
	}

	fmt.Println("任务1执行时间: ", time.Since(start), "总共计算：", count, "个点")
}

func (c *Calculator) calculateCase2() {
	var start = time.Now()
	var deltaT = c.calculateTimeStep()
	var count = 0
	for k := 0; k < ZLength/ZStep; k++ {
		// 先计算点，再计算外表面，再计算里面的点
		c.calculatePointRT(deltaT, k)
		count++
		for i := Length / XStep / 2; i < Length/XStep-1; i++ {
			c.calculatePointTA(deltaT, i, k)
			count++
		}
		for j := Width / YStep / 2; j < Width/YStep-1; j++ {
			c.calculatePointRA(deltaT, j, k)
			count++
		}
		for j := Width/YStep - 1 - c.EdgeWidth; j < Width/YStep-1; j++ {
			for i := Length/XStep - 1 - c.EdgeWidth; i < Length/XStep-1; i++ {
				c.calculatePointIN(deltaT, i, j, k)
				count++
			}
		}
		for j := Width / YStep / 2; j < Width/YStep-1-c.EdgeWidth; j++ {
			for i := Length/XStep - 1 - c.EdgeWidth; i < Length/XStep-1; i++ {
				c.calculatePointIN(deltaT, i, j, k)
				count++
			}
		}
		for j := Width/YStep - 1 - c.EdgeWidth; j < Width/YStep-1; j++ {
			for i := Length / XStep / 2; i < Length/XStep-1-c.EdgeWidth; i = i + 1 {
				c.calculatePointIN(deltaT, i, j, k)
				count++
			}
		}
		for j := Width / YStep / 2; j < Width/YStep-1-c.EdgeWidth; j = j + c.Step {
			for i := Length / XStep / 2; i < Length/XStep-1-c.EdgeWidth; i = i + c.Step {
				c.calculatePointIN(deltaT, i, j, k)
				count++
			}
		}
		c.alternating = !c.alternating
	}
	fmt.Println("任务2执行时间: ", time.Since(start), "总共计算：", count, "个点")
}

func (c *Calculator) calculateCase3() {
	var start = time.Now()
	var deltaT = c.calculateTimeStep()
	var count = 0
	for k := 0; k < ZLength/ZStep; k++ {
		// 先计算点，再计算外表面，再计算里面的点
		c.calculatePointRB(deltaT, k)
		count++
		for i := Length / XStep / 2; i < Length/XStep-1; i++ {
			c.calculatePointBA(deltaT, i, k)
			count++
		}
		for j := 1; j < Width/YStep/2; j++ {
			c.calculatePointRA(deltaT, j, k)
			count++
		}
		for j := 1; j < 1+c.EdgeWidth; j++ {
			for i := Length/XStep - 1 - c.EdgeWidth; i < Length/XStep-1; i++ {
				c.calculatePointIN(deltaT, i, j, k)
				count++
			}
		}
		for j := 1 + c.EdgeWidth; j < Width/YStep/2; j++ {
			for i := Length/XStep - 1 - c.EdgeWidth; i < Length/XStep-1; i++ {
				c.calculatePointIN(deltaT, i, j, k)
				count++
			}
		}
		for j := 1; j < 1+c.EdgeWidth; j++ {
			for i := Length / XStep / 2; i < Length/XStep-1-c.EdgeWidth; i++ {
				c.calculatePointIN(deltaT, i, j, k)
				count++
			}
		}
		for j := 1 + c.EdgeWidth; j < Width/YStep/2; j = j + c.Step {
			for i := Length / XStep / 2; i < Length/XStep-1-c.EdgeWidth; i = i + c.Step {
				c.calculatePointIN(deltaT, i, j, k)
				count++
			}
		}
		c.alternating = !c.alternating
	}
	fmt.Println("任务3执行时间: ", time.Since(start), "总共计算：", count, "个点")
}

func (c *Calculator) calculateCase4() {
	var start = time.Now()
	var deltaT = c.calculateTimeStep()
	var count = 0
	for k := 0; k < ZLength/ZStep; k++ {
		// 先计算点，再计算外表面，再计算里面的点
		c.calculatePointLB(deltaT, k)
		count++
		for i := 1; i < Length/XStep/2; i++ {
			c.calculatePointBA(deltaT, i, k)
			count++
		}
		for j := 1; j < Width/YStep/2; j++ {
			c.calculatePointLA(deltaT, j, k)
			count++
		}
		for j := 1; j < 1+c.EdgeWidth; j++ {
			for i := 1; i < 1+c.EdgeWidth; i++ {
				c.calculatePointIN(deltaT, i, j, k)
				count++
			}
		}
		for j := 1 + c.EdgeWidth; j < Width/YStep/2; j++ {
			for i := 1; i < 1+c.EdgeWidth; i++ {
				c.calculatePointIN(deltaT, i, j, k)
				count++
			}
		}
		for j := 1; j < 1+c.EdgeWidth; j++ {
			for i := 1 + c.EdgeWidth; i < Length/XStep/2; i++ {
				c.calculatePointIN(deltaT, i, j, k)
				count++
			}
		}
		for j := 1 + c.EdgeWidth; j < Width/YStep/2; j = j + c.Step {
			for i := 1 + c.EdgeWidth; i < Length/XStep/2; i = i + c.Step {
				c.calculatePointIN(deltaT, i, j, k)
				count++
			}
		}
		c.alternating = !c.alternating
	}
	fmt.Println("任务4执行时间: ", time.Since(start), "总共计算：", count, "个点")
}

func (c *Calculator) CalculateConcurrently() {
	var start = time.Now()
	var wg = sync.WaitGroup{}
	wg.Add(4)
	go func() {
		c.calculateCase1()
		wg.Done()
	}()
	go func() {
		c.calculateCase2()
		wg.Done()
	}()
	go func() {
		c.calculateCase3()
		wg.Done()
	}()
	go func() {
		c.calculateCase4()
		wg.Done()
	}()
	wg.Wait()
	fmt.Println("并行计算时间：", time.Since(start))
}

func (c *Calculator) Calculate() {
	// 四个核心一起计算
	c.CalculateConcurrently()
	// 一个核心计算
	c.CalculateSerially()
}
