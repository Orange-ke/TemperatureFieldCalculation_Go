package calculator

import (
	"fmt"
	"time"
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
	Field = [ZLength / ZStep][Width / YStep][Length / XStep]float32{}

	upUp    = [Width / YStep][Length / XStep]float32{}
	upLeft  = [Width / YStep][upLength / stepZ]float32{}
	upRight = [Width / YStep][upLength / stepZ]float32{}
	upFront = [Length / XStep][upLength / stepZ]float32{}
	upBack  = [Length / XStep][upLength / stepZ]float32{}

	arcLeft  = [Width / YStep][arcLength / stepZ]float32{}
	arcRight = [Width / YStep][arcLength / stepZ]float32{}
	arcFront = [Length / XStep][arcLength / stepZ]float32{}
	arcBack  = [Length / XStep][arcLength / stepZ]float32{}

	downDown  = [Width / YStep][Length / XStep]float32{}
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
	Up    *[Width / YStep][Length / XStep]float32    `json:"up"`
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
	Down  *[Width / YStep][Length / XStep]float32    `json:"down"`
}

// 从1/4构建整个切片
func (c *Calculator) BuildData() *TemperatureData {
	upSides := &UpSides{
		Up:    &upUp,
		Left:  &upLeft,
		Right: &upRight,
		Front: &upFront,
		Back:  &upBack,
	}
	arcSides := &ArcSides{
		Left:  &arcLeft,
		Right: &arcRight,
		Front: &arcFront,
		Back:  &arcBack,
	}
	downSides := &DownSides{
		Down:  &downDown,
		Left:  &downLeft,
		Right: &downRight,
		Front: &downFront,
		Back:  &downBack,
	}
	temperatureData := &TemperatureData{
		Up:   upSides,
		Arc:  arcSides,
		Down: downSides,
	}
	// todo
	ThermalField := &ThermalFieldStruct{
		Field: &Field,
	}

	if c.ThermalField.IsFull {
		fmt.Println("切片已充满......")
		l := ThermalFieldCopyPtr.End - ThermalFieldCopyPtr.Start
		for z := ThermalFieldCopyPtr.Start; z < ThermalFieldCopyPtr.End; z++ {
			Field[l-(ThermalFieldCopyPtr.End-z)] = ThermalFieldCopyPtr.Field[z]
		}

		for z := c.ThermalField.Start; z < c.ThermalField.End; z++ {
			Field[z] = c.ThermalField.Field[z]
		}

		ThermalField.Start = 0
		ThermalField.End = ZLength / ZStep
		ThermalField.IsFull = true
	} else {
		fmt.Println("切片未充满......", c.ThermalField.IsCopy)
		start := c.ThermalField.Start
		for z := c.ThermalField.Start; z < c.ThermalField.End; z++ {
			Field[z-start] = c.ThermalField.Field[z]
		}
		l := c.ThermalField.End - c.ThermalField.Start
		fmt.Println("切片数为：", l)
		ThermalField.Start = 0
		ThermalField.End = l
	}

	// todo 处理拉尾坯模式

	c.buildDataHelper(ThermalField, temperatureData)
	temperatureData.Start = ThermalField.Start
	temperatureData.End = ThermalField.End
	temperatureData.IsFull = ThermalField.IsFull
	return temperatureData
}

func (c *Calculator) buildDataHelper(ThermalField *ThermalFieldStruct, temperatureData *TemperatureData) {
	startTime := time.Now()
	// up
	for y := Width/YStep - 1; y >= 0; y -= stepY {
		for x := Length/XStep - 1; x >= 0; x -= stepX {
			temperatureData.Up.Up[Width/YStep/2+y/stepY][Length/XStep/2+x/stepX] = ThermalField.Field[ThermalField.Start][y][x]
			temperatureData.Up.Up[Width/YStep/2-1-y/stepY][Length/XStep/2-1-x/stepX] = ThermalField.Field[ThermalField.Start][y][x]
			temperatureData.Up.Up[Width/YStep/2+y/stepY][Length/XStep/2-1-x/stepX] = ThermalField.Field[ThermalField.Start][y][x]
			temperatureData.Up.Up[Width/YStep/2-1-y/stepY][Length/XStep/2+x/stepX] = ThermalField.Field[ThermalField.Start][y][x]
		}
	}
	zStart := ThermalField.Start
	zEnd := upLength
	if ThermalField.End < zEnd {
		zEnd = ThermalField.End
	}
	for y := Width/YStep - 1; y >= 0; y -= stepY {
		for x := zStart; x < zEnd; x += stepZ {
			temperatureData.Up.Left[Width/YStep/2+y/stepY][x/stepZ] = ThermalField.Field[x][y][Length/XStep-1]
			temperatureData.Up.Left[Width/YStep/2-1-y/stepY][x/stepZ] = ThermalField.Field[x][y][Length/XStep-1]
		}
	}
	for y := Width/YStep - 1; y >= 0; y -= stepY {
		for x := zStart; x < zEnd; x += stepZ {
			temperatureData.Up.Right[Width/YStep/2+y/stepY][x/stepZ] = ThermalField.Field[x][y][Length/XStep-1]
			temperatureData.Up.Right[Width/YStep/2-1-y/stepY][x/stepZ] = ThermalField.Field[x][y][Length/XStep-1]
		}
	}
	for y := Length/XStep - 1; y >= 0; y -= stepX {
		for x := zStart; x < zEnd; x += stepZ {
			temperatureData.Up.Front[Length/XStep/2+y/stepX][x/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
			temperatureData.Up.Front[Length/XStep/2-y/stepX-1][x/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
		}
	}
	for y := Length/XStep - 1; y >= 0; y -= stepX {
		for x := zStart; x < zEnd; x += stepZ {
			temperatureData.Up.Back[Length/XStep/2+y/stepX][x/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
			temperatureData.Up.Back[Length/XStep/2-y/stepX-1][x/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
		}
	}

	zStart = upLength
	zEnd = upLength + arcLength
	if ThermalField.End < zEnd {
		zEnd = ThermalField.End
	}
	for y := Width/YStep - 1; y >= 0; y -= stepY {
		for x := zStart; x < zEnd; x += stepZ {
			temperatureData.Arc.Left[Width/YStep/2+y/stepY][(x-zStart)/stepZ] = ThermalField.Field[x][y][0]
			temperatureData.Arc.Left[Width/YStep/2-1-y/stepY][(x-zStart)/stepZ] = ThermalField.Field[x][y][0]
		}
	}
	for y := Width/YStep - 1; y >= 0; y -= stepY {
		for x := zStart; x < zEnd; x += stepZ {
			temperatureData.Arc.Right[Width/YStep/2+y/stepY][(x-zStart)/stepZ] = ThermalField.Field[x][y][0]
			temperatureData.Arc.Right[Width/YStep/2-1-y/stepY][(x-zStart)/stepZ] = ThermalField.Field[x][y][0]
		}
	}
	for y := Length/XStep - 1; y >= 0; y -= stepX {
		for x := zStart; x < zEnd; x += stepZ {
			temperatureData.Arc.Front[Length/XStep/2+y/stepX][(x-zStart)/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
			temperatureData.Arc.Front[Length/XStep/2-y/stepX-1][(x-zStart)/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
		}
	}
	for y := Length/XStep - 1; y >= 0; y -= stepX {
		for x := zStart; x < zEnd; x += stepZ {
			temperatureData.Arc.Back[Length/XStep/2+y/stepX][(x-zStart)/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
			temperatureData.Arc.Back[Length/XStep/2-y/stepX-1][(x-zStart)/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
		}
	}

	zStart = upLength + arcLength
	zEnd = upLength + arcLength + downLength
	if ThermalField.End < zEnd {
		zEnd = ThermalField.End
	}
	for y := Width/YStep - 1; y >= 0; y -= stepY {
		for x := Length/XStep - 1; x >= 0; x -= stepX {
			temperatureData.Down.Down[Width/YStep/2+y/stepY][Length/XStep/2+x/stepX] = ThermalField.Field[ThermalField.End-1][y][x]
			temperatureData.Down.Down[Width/YStep/2-1-y/stepY][Length/XStep/2-1-x/stepX] = ThermalField.Field[ThermalField.End-1][y][x]
			temperatureData.Down.Down[Width/YStep/2+y/stepY][Length/XStep/2-1-x/stepX] = ThermalField.Field[ThermalField.End-1][y][x]
			temperatureData.Down.Down[Width/YStep/2-1-y/stepY][Length/XStep/2+x/stepX] = ThermalField.Field[ThermalField.End-1][y][x]
		}
	}
	for y := Width/YStep - 1; y >= 0; y -= stepY {
		for x := zStart; x < zEnd; x += stepZ {
			temperatureData.Down.Left[Width/YStep/2+y/stepY][(x-zStart)/stepZ] = ThermalField.Field[x][y][0]
			temperatureData.Down.Left[Width/YStep/2-y/stepY-1][(x-zStart)/stepZ] = ThermalField.Field[x][y][0]
		}
	}
	for y := Width/YStep - 1; y >= 0; y -= stepY {
		for x := zStart; x < zEnd; x += stepZ {
			temperatureData.Down.Right[Width/YStep/2+y/stepY][(x-zStart)/stepZ] = ThermalField.Field[x][y][0]
			temperatureData.Down.Right[Width/YStep/2-y/stepY-1][(x-zStart)/stepZ] = ThermalField.Field[x][y][0]
		}
	}
	for y := Length/XStep - 1; y >= 0; y -= stepX {
		for x := zStart; x < zEnd; x += stepZ {
			temperatureData.Down.Front[Length/XStep/2+y/stepX][(x-zStart)/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
			temperatureData.Down.Front[Length/XStep/2-y/stepX-1][(x-zStart)/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
		}
	}
	for y := Length/XStep - 1; y >= 0; y -= stepX {
		for x := zStart; x < zEnd; x += stepZ {
			temperatureData.Down.Back[Length/XStep/2+y/stepX][(x-zStart)/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
			temperatureData.Down.Back[Length/XStep/2-y/stepX-1][(x-zStart)/stepZ] = ThermalField.Field[x][Width/YStep-1][y]
		}
	}
	fmt.Println("build data cost: ", time.Since(startTime))
	// temperatureData
}
