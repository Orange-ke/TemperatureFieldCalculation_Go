package calculator

import (
	"fmt"
	"time"
)


const (
	upLength   = 500
	downLength = 500
	arcLength  = 3000

	stepX = 1
	stepY = 1
	stepZ = 10
)

type TemperatureData struct {
	Start  int       `json:"start"`   // 切片开始位置
	End    int       `json:"end"`     // 切片结束位置
	IsFull bool      `json:"is_full"` // 切片是否充满铸机
	Up     UpSides   `json:"up"`
	Arc    ArcSides  `json:"arc"`
	Down   DownSides `json:"down"`
}

type UpSides struct {
	Up    [Width / YStep / stepY][Length / XStep / stepX]float32 `json:"up"`
	Left  [Width / YStep / stepY][upLength / stepZ]float32                             `json:"left"`
	Right [Width / YStep / stepY][upLength / stepZ]float32                             `json:"right"`
	Front [Length / XStep / stepX][upLength / stepZ]float32                            `json:"front"`
	Back  [Length / XStep / stepX][upLength / stepZ]float32                            `json:"back"`
}

type ArcSides struct {
	Left  [Width / YStep / stepY][arcLength / stepZ]float32  `json:"left"`
	Right [Width / YStep / stepY][arcLength / stepZ]float32  `json:"right"`
	Front [Length / XStep / stepX][arcLength / stepZ]float32 `json:"front"`
	Back  [Length / XStep / stepX][arcLength / stepZ]float32 `json:"back"`
}

type DownSides struct {
	Left  [Width / YStep / stepY][upLength / stepZ]float32                             `json:"left"`
	Right [Width / YStep / stepY][upLength / stepZ]float32                             `json:"right"`
	Front [Length / XStep / stepX][upLength / stepZ]float32                            `json:"front"`
	Back  [Length / XStep / stepX][upLength / stepZ]float32                            `json:"back"`
	Down  [Width / YStep / stepY][Length / XStep / stepX]float32 `json:"down"`
}

func (c *Calculator) BuildData() TemperatureData {
	start := time.Now()
	upSides := UpSides{}
	for y := 0; y < Width/YStep; y += stepY {
		for x := 0; x < Length/XStep; x += stepX {
			upSides.Up[y/stepY][x/stepX] = c.ThermalField[0][y][x]
		}
	}
	for y := 0; y < Width/YStep; y += stepY {
		for x := 0; x < upLength; x += stepZ {
			upSides.Left[y/stepY][x/stepZ] = c.ThermalField[x][y][0]
		}
	}
	for y := 0; y < Width/YStep; y += stepY {
		for x := 0; x < upLength; x += stepZ {
			upSides.Right[y/stepY][x/stepZ] = c.ThermalField[x][y][Length/XStep-1]
		}
	}
	for y := 0; y < Length/XStep; y += stepX {
		for x := 0; x < upLength; x += stepZ {
			upSides.Front[y/stepX][x/stepZ] = c.ThermalField[x][Width/YStep-1][y]
		}
	}
	for y := 0; y < Length/XStep; y += stepX {
		for x := 0; x < upLength; x += stepZ {
			upSides.Back[y/stepX][x/stepZ] = c.ThermalField[x][0][y]
		}
	}

	arcSides := ArcSides{}
	zStart := upLength
	zEnd := upLength + arcLength
	for y := 0; y < Width/YStep; y += stepY {
		for x := zStart; x < zEnd; x += stepZ {
			arcSides.Left[y/stepY][(x-zStart)/stepZ] = c.ThermalField[x][y][0]
		}
	}
	for y := 0; y < Width/YStep; y += stepY {
		for x := zStart; x < zEnd; x += stepZ {
			arcSides.Right[y/stepY][(x-zStart)/stepZ] = c.ThermalField[x][y][Length/XStep-1]
		}
	}
	for y := 0; y < Length/XStep; y += stepX {
		for x := zStart; x < zEnd; x += stepZ {
			arcSides.Front[y/stepX][(x-zStart)/stepZ] = c.ThermalField[x][Width/YStep-1][y]
		}
	}
	for y := 0; y < Length/XStep; y += stepX {
		for x := zStart; x < zEnd; x += stepZ {
			arcSides.Back[y/stepX][(x-zStart)/stepZ] = c.ThermalField[x][0][y]
		}
	}

	downSides := DownSides{}
	zStart = upLength + arcLength
	zEnd = upLength + arcLength + downLength
	for y := 0; y < Width/YStep; y += stepY {
		for x := 0; x < Length/XStep; x += stepX {
			downSides.Down[y/stepY][x/stepX] = c.ThermalField[zEnd-1][y][x]
		}
	}
	for y := 0; y < Width/YStep; y += stepY {
		for x := zStart; x < zEnd; x += stepZ {
			downSides.Left[y/stepY][(x-zStart)/stepZ] = c.ThermalField[x][y][0]
		}
	}
	for y := 0; y < Width/YStep; y += stepY {
		for x := zStart; x < zEnd; x += stepZ {
			downSides.Right[y/stepY][(x-zStart)/stepZ] = c.ThermalField[x][y][Length/XStep-1]
		}
	}
	for y := 0; y < Length/XStep; y += stepX {
		for x := zStart; x < zEnd; x += stepZ {
			downSides.Front[y/stepX][(x-zStart)/stepZ] = c.ThermalField[x][Width/YStep-1][y]
		}
	}
	for y := 0; y < Length/XStep; y += stepX {
		for x := zStart; x < zEnd; x += stepZ {
			downSides.Back[y/stepX][(x-zStart)/stepZ] = c.ThermalField[x][0][y]
		}
	}
	fmt.Println("build data cost: ", time.Since(start))
	return TemperatureData{
		Start:  0,
		End:    400,
		IsFull: true,
		Up:     upSides,
		Arc:    arcSides,
		Down:   downSides,
	}
}
