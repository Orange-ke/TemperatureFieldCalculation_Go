package calculator

import "lz/model"

const (
	ArrayLength = 1550 / 5
)

var (
	Density  [ArrayLength]float32 // 密度
	Enthalpy [ArrayLength]float32 // 焓
	Lambda   [ArrayLength]float32 // 导热系数
	HEff     [ArrayLength]float32 // 综合换热系数, 注意：简单处理了！
	Q        [ArrayLength]float32 // 热流密度, 注意：简单处理了
	C        [ArrayLength]float32 // 比热容
)

type ThermalFieldStruct struct {
	Start  int
	End    int
	Field  *[ZLength / ZStep]model.ItemType
	IsFull bool
	IsTail bool
	IsCopy bool
}