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
