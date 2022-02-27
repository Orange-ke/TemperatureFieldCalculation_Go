package calculator

import "lz/model"

type ThermalFieldStruct struct {
	Start  int
	End    int
	Field  *[model.ZLength / model.ZStep]model.ItemType
	IsFull bool
	IsTail bool
}

type SlicePushDataStruct struct {
	Slice   *[model.Width / model.YStep * 2][model.Length / model.XStep * 2]float32 `json:"slice"`
	Marks   map[int]string                                                          `json:"marks"`
	Start   int                                                                     `json:"start"`
	End     int                                                                     `json:"end"`
	Current int                                                                     `json:"current"`
}

const (
	ArrayLength = 1600
)

type parameter struct {
	Density  [ArrayLength]float32 // 密度
	Enthalpy [ArrayLength]float32 // 焓
	Lambda   [ArrayLength]float32 // 导热系数
	HEff     [ArrayLength]float32 // 综合换热系数, 注意：简单处理了 todo 需要更新，以函数的方式进行调用， 结晶体，棍子，不同钢种
	Q        [ArrayLength]float32 // 热流密度, 注意：简单处理了 todo 需要更新，以函数的方式进行调用
	C        [ArrayLength]float32 // 比热容
	GetHeff  func(T float32, parameter *parameter) float32
	GetQ     func(T float32, parameter *parameter) float32
}

type coolerConfig struct {
	StartTemperature       float32
	NarrowSurfaceIn        float32
	NarrowSurfaceOut       float32
	WideSurfaceIn          float32
	WideSurfaceOut         float32
	SprayTemperature       float32
	RollerWaterTemperature float32
}
