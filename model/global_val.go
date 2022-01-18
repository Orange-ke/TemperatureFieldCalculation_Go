package model

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
)

// 元素类型
type ItemType [Width / YStep][Length / XStep]float32
