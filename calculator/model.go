package calculator

import "lz/model"

type ThermalFieldStruct struct {
	Start  int
	End    int
	Field  []model.ItemType
	IsFull bool
	IsTail bool
}

type SlicePushDataStruct struct {
	Slice   [][]float32    `json:"slice"`
	Start   int            `json:"start"`
	End     int            `json:"end"`
	Current int            `json:"current"`
}
