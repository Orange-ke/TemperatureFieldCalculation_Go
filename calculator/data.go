package calculator

import (
	"lz/model"
)

const (
	upLength   = 500
	downLength = 500
	arcLength  = 3000

	stepX = 2
	stepY = 2
	stepZ = 10
)

var (
	Field = [model.ZLength / model.ZStep]model.ItemType{}

	upUp    = model.ItemType{}
	upLeft  = [model.Width / model.YStep][upLength / stepZ]float32{}
	upRight = [model.Width / model.YStep][upLength / stepZ]float32{}
	upFront = [model.Length / model.XStep][upLength / stepZ]float32{}
	upBack  = [model.Length / model.XStep][upLength / stepZ]float32{}

	arcLeft  = [model.Width / model.YStep][arcLength / stepZ]float32{}
	arcRight = [model.Width / model.YStep][arcLength / stepZ]float32{}
	arcFront = [model.Length / model.XStep][arcLength / stepZ]float32{}
	arcBack  = [model.Length / model.XStep][arcLength / stepZ]float32{}

	downDown  = model.ItemType{}
	downLeft  = [model.Width / model.YStep][upLength / stepZ]float32{}
	downRight = [model.Width / model.YStep][upLength / stepZ]float32{}
	downFront = [model.Length / model.XStep][upLength / stepZ]float32{}
	downBack  = [model.Length / model.XStep][upLength / stepZ]float32{}
)

type TemperatureData struct {
	Start  int        `json:"start"`   // 切片开始位置
	End    int        `json:"end"`     // 切片结束位置
	IsFull bool       `json:"is_full"` // 切片是否充满铸机
	IsTail bool       `json:"is_tail"` // 是否拉尾坯
	Up     *UpSides   `json:"up"`
	Arc    *ArcSides  `json:"arc"`
	Down   *DownSides `json:"down"`
}

type UpSides struct {
	Up    *model.ItemType                                        `json:"up"`
	Left  *[model.Width / model.YStep][upLength / stepZ]float32  `json:"left"`
	Right *[model.Width / model.YStep][upLength / stepZ]float32  `json:"right"`
	Front *[model.Length / model.XStep][upLength / stepZ]float32 `json:"front"`
	Back  *[model.Length / model.XStep][upLength / stepZ]float32 `json:"back"`
}

type ArcSides struct {
	Left  *[model.Width / model.YStep][arcLength / stepZ]float32  `json:"left"`
	Right *[model.Width / model.YStep][arcLength / stepZ]float32  `json:"right"`
	Front *[model.Length / model.XStep][arcLength / stepZ]float32 `json:"front"`
	Back  *[model.Length / model.XStep][arcLength / stepZ]float32 `json:"back"`
}

type DownSides struct {
	Left  *[model.Width / model.YStep][upLength / stepZ]float32  `json:"left"`
	Right *[model.Width / model.YStep][upLength / stepZ]float32  `json:"right"`
	Front *[model.Length / model.XStep][upLength / stepZ]float32 `json:"front"`
	Back  *[model.Length / model.XStep][upLength / stepZ]float32 `json:"back"`
	Down  *model.ItemType                                        `json:"down"`
}
