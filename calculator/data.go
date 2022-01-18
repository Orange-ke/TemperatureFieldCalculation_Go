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
	Field = [ZLength / ZStep]model.ItemType{}

	upUp    = model.ItemType{}
	upLeft  = [Width / YStep][upLength / stepZ]float32{}
	upRight = [Width / YStep][upLength / stepZ]float32{}
	upFront = [Length / XStep][upLength / stepZ]float32{}
	upBack  = [Length / XStep][upLength / stepZ]float32{}

	arcLeft  = [Width / YStep][arcLength / stepZ]float32{}
	arcRight = [Width / YStep][arcLength / stepZ]float32{}
	arcFront = [Length / XStep][arcLength / stepZ]float32{}
	arcBack  = [Length / XStep][arcLength / stepZ]float32{}

	downDown  = model.ItemType{}
	downLeft  = [Width / YStep][upLength / stepZ]float32{}
	downRight = [Width / YStep][upLength / stepZ]float32{}
	downFront = [Length / XStep][upLength / stepZ]float32{}
	downBack  = [Length / XStep][upLength / stepZ]float32{}
)

type TemperatureData struct {
	Start  int        `json:"start"`   // 切片开始位置
	End    int        `json:"end"`     // 切片结束位置
	IsFull bool       `json:"is_full"` // 切片是否充满铸机
	Up     *UpSides   `json:"up"`
	Arc    *ArcSides  `json:"arc"`
	Down   *DownSides `json:"down"`
}

type UpSides struct {
	Up    *model.ItemType    `json:"up"`
	Left  *[Width / YStep][upLength / stepZ]float32  `json:"left"`
	Right *[Width / YStep][upLength / stepZ]float32  `json:"right"`
	Front *[Length / XStep][upLength / stepZ]float32 `json:"front"`
	Back  *[Length / XStep][upLength / stepZ]float32 `json:"back"`
}

type ArcSides struct {
	Left  *[Width / YStep][arcLength / stepZ]float32  `json:"left"`
	Right *[Width / YStep][arcLength / stepZ]float32  `json:"right"`
	Front *[Length / XStep][arcLength / stepZ]float32 `json:"front"`
	Back  *[Length / XStep][arcLength / stepZ]float32 `json:"back"`
}

type DownSides struct {
	Left  *[Width / YStep][upLength / stepZ]float32  `json:"left"`
	Right *[Width / YStep][upLength / stepZ]float32  `json:"right"`
	Front *[Length / XStep][upLength / stepZ]float32 `json:"front"`
	Back  *[Length / XStep][upLength / stepZ]float32 `json:"back"`
	Down  *model.ItemType    `json:"down"`
}
