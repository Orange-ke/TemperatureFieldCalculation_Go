package calculator

import (
	"lz/casting_machine"
)

type ThermalFieldStruct struct {
	Start  int
	End    int
	Field  *[casting_machine.ZLength / casting_machine.ZStep]casting_machine.ItemType
	IsFull bool
	IsTail bool
}

type SlicePushDataStruct struct {
	Slice   *[casting_machine.Width / casting_machine.YStep * 2][casting_machine.Length / casting_machine.XStep * 2]float32 `json:"slice"`
	Marks   map[int]string                                                          `json:"marks"`
	Start   int                                                                     `json:"start"`
	End     int                                                                     `json:"end"`
	Current int                                                                     `json:"current"`
}
