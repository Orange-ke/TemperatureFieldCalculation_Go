package calculator

import (
	"lz/model"
	"strconv"
)

var (
	upUp    = model.ItemType{}
	downDown  = model.ItemType{}
)

type TemperatureFieldData struct {
	Start  int        `json:"start"`   // 切片开始位置
	End    int        `json:"end"`     // 切片结束位置
	IsFull bool       `json:"is_full"` // 切片是否充满铸机
	IsTail bool       `json:"is_tail"` // 是否拉尾坯
	Up     *UpSides   `json:"up"`
	Arc    *ArcSides  `json:"arc"`
	Down   *DownSides `json:"down"`
}

type UpSides struct {
	Up    *model.ItemType             `json:"up"`
	Left  [][UpLength / StepZ]float32 `json:"left"`
	Right [][UpLength / StepZ]float32 `json:"right"`
	Front [][UpLength / StepZ]float32 `json:"front"`
	Back  [][UpLength / StepZ]float32 `json:"back"`
}

type ArcSides struct {
	Left  [][ArcLength / StepZ]float32 `json:"left"`
	Right [][ArcLength / StepZ]float32 `json:"right"`
	Front [][ArcLength / StepZ]float32 `json:"front"`
	Back  [][ArcLength / StepZ]float32 `json:"back"`
}

type DownSides struct {
	Left  [][UpLength / StepZ]float32 `json:"left"`
	Right [][UpLength / StepZ]float32 `json:"right"`
	Front [][UpLength / StepZ]float32 `json:"front"`
	Back  [][UpLength / StepZ]float32 `json:"back"`
	Down  *model.ItemType             `json:"down"`
}

type PushData struct {
	Top    Encoding
	Arc    Encoding
	Bottom Encoding
}

type Encoding struct {
	Start []byte
	Data  [][]byte
	Max   []byte
}

type Decoding struct {
	Start []int
	Data  [][]int
	Max   []int
}

func (c *calculatorWithArrDeque) BuildData() *TemperatureFieldData {
	upSides := &UpSides{
		Up:    &upUp,
		Left:  make([][UpLength / StepZ]float32, Width/YStep),
		Right: make([][UpLength / StepZ]float32, Width/YStep),
		Front: make([][UpLength / StepZ]float32, Length/XStep),
		Back:  make([][UpLength / StepZ]float32, Length/XStep),
	}
	arcSides := &ArcSides{
		Left:  make([][ArcLength / StepZ]float32, Width/YStep),
		Right: make([][ArcLength / StepZ]float32, Width/YStep),
		Front: make([][ArcLength / StepZ]float32, Length/XStep),
		Back:  make([][ArcLength / StepZ]float32, Length/XStep),
	}
	downSides := &DownSides{
		Down:  &downDown,
		Left:  make([][DownLength / StepZ]float32, Width/YStep),
		Right: make([][DownLength / StepZ]float32, Width/YStep),
		Front: make([][DownLength / StepZ]float32, Length/XStep),
		Back:  make([][DownLength / StepZ]float32, Length/XStep),
	}
	temperatureData := &TemperatureFieldData{
		Up:   upSides,
		Arc:  arcSides,
		Down: downSides,
	}

	ThermalField := &ThermalFieldStruct{
		Field: make([]model.ItemType, ZLength/ZStep),
	}

	z := 0
	//for i := 0; i < 2000; i++ {
	//	c.Field.RemoveLast()
	//}
	c.Field.Traverse(func(_ int, item *model.ItemType) {
		ThermalField.Field[z] = *item
		z++
	})

	if c.isFull {
		ThermalField.Start = 0
		ThermalField.End = ZLength / model.ZStep
		ThermalField.IsFull = true
	} else {
		ThermalField.Start = 0
		ThermalField.End = z
	}

	if c.isTail {
		ThermalField.IsTail = true
	}
	//fmt.Println("BuildData 温度场的长度：", z)
	//if !c.Field.IsEmpty() {
	//	for i := casting_machine.Width/casting_machine.YStep - 1; i > casting_machine.Width/casting_machine.YStep-6; i-- {
	//		for j := casting_machine.Length/casting_machine.XStep - 5; j <= casting_machine.Length/casting_machine.XStep-1; j++ {
	//			fmt.Print(Field[z-1][i][j], " ")
	//		}
	//		fmt.Print(i, "build data")
	//		fmt.Println()
	//	}
	//}

	buildDataHelper(ThermalField, temperatureData)
	temperatureData.Start = ThermalField.Start
	temperatureData.End = ThermalField.End
	temperatureData.IsFull = ThermalField.IsFull
	temperatureData.IsTail = ThermalField.IsTail
	return temperatureData
}

// 构建温度场push data
func buildDataHelper(ThermalField *ThermalFieldStruct, temperatureData *TemperatureFieldData) {
	// 跳过为空的切片
	for ThermalField.Field[ThermalField.Start][0][0] == -1 {
		ThermalField.Start++
		if ThermalField.Start == ZLength/model.ZStep {
			return
		}
	}
	//startTime := time.Now()
	// up
	for y := Width/YStep - 1; y >= 0; y -= StepY {
		for x := Length/XStep - 1; x >= 0; x -= StepX {
			temperatureData.Up.Up[Width/YStep/2+y/StepY][Length/XStep/2+x/StepX] = ThermalField.Field[ThermalField.Start][y][x]
			temperatureData.Up.Up[(Width/YStep-1)/2-y/StepY][(Length/XStep-1)/2-x/StepX] = ThermalField.Field[ThermalField.Start][y][x]
			temperatureData.Up.Up[Width/YStep/2+y/StepY][(Length/XStep-1)/2-x/StepX] = ThermalField.Field[ThermalField.Start][y][x]
			temperatureData.Up.Up[(Width/YStep-1)/2-y/StepY][Length/XStep/2+x/StepX] = ThermalField.Field[ThermalField.Start][y][x]
		}
	}
	start := 0
	zStart := ThermalField.Start
	zEnd := UpLength
	if ThermalField.End < zEnd {
		zEnd = ThermalField.End
	}
	for y := Width/YStep - 1; y >= 0; y -= StepY {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Up.Left[Width/YStep/2+y/StepY][x/StepZ] = ThermalField.Field[x][y][Length/XStep-1]
			temperatureData.Up.Left[(Width/YStep-1)/2-y/StepY][x/StepZ] = ThermalField.Field[x][y][Length/XStep-1]
		}
	}
	for y := Width/YStep - 1; y >= 0; y -= StepY {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Up.Right[Width/YStep/2+y/StepY][x/StepZ] = ThermalField.Field[x][y][Length/XStep-1]
			temperatureData.Up.Right[(Width/YStep-1)/2-y/StepY][x/StepZ] = ThermalField.Field[x][y][Length/XStep-1]
		}
	}
	for y := Length/XStep - 1; y >= 0; y -= StepX {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Up.Front[Length/XStep/2+y/StepX][x/StepZ] = ThermalField.Field[x][Width/YStep-1][y]
			temperatureData.Up.Front[(Length/XStep-1)/2-y/StepX][x/StepZ] = ThermalField.Field[x][Width/YStep-1][y]
		}
	}
	for y := Length/XStep - 1; y >= 0; y -= StepX {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Up.Back[Length/XStep/2+y/StepX][x/StepZ] = ThermalField.Field[x][Width/YStep-1][y]
			temperatureData.Up.Back[(Length/XStep-1)/2-y/StepX][x/StepZ] = ThermalField.Field[x][Width/YStep-1][y]
		}
	}

	start = UpLength
	zStart = max(UpLength, ThermalField.Start)
	zEnd = UpLength + ArcLength
	if ThermalField.End < zEnd {
		zEnd = ThermalField.End
	}
	for y := Width/YStep - 1; y >= 0; y -= StepY {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Arc.Left[Width/YStep/2+y/StepY][(x-start)/StepZ] = ThermalField.Field[x][y][Length/XStep-1]
			temperatureData.Arc.Left[(Width/YStep-1)/2-y/StepY][(x-start)/StepZ] = ThermalField.Field[x][y][Length/XStep-1]
		}
	}
	for y := Width/YStep - 1; y >= 0; y -= StepY {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Arc.Right[Width/YStep/2+y/StepY][(x-start)/StepZ] = ThermalField.Field[x][y][Length/XStep-1]
			temperatureData.Arc.Right[(Width/YStep-1)/2-y/StepY][(x-start)/StepZ] = ThermalField.Field[x][y][Length/XStep-1]
		}
	}
	for y := Length/XStep - 1; y >= 0; y -= StepX {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Arc.Front[Length/XStep/2+y/StepX][(x-start)/StepZ] = ThermalField.Field[x][Width/YStep-1][y]
			temperatureData.Arc.Front[(Length/XStep-1)/2-y/StepX][(x-start)/StepZ] = ThermalField.Field[x][Width/YStep-1][y]
		}
	}
	for y := Length/XStep - 1; y >= 0; y -= StepX {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Arc.Back[Length/XStep/2+y/StepX][(x-start)/StepZ] = ThermalField.Field[x][Width/YStep-1][y]
			temperatureData.Arc.Back[(Length/XStep-1)/2-y/StepX][(x-start)/StepZ] = ThermalField.Field[x][Width/YStep-1][y]
		}
	}

	start = UpLength + ArcLength
	zStart = max(UpLength+ArcLength, ThermalField.Start)
	zEnd = UpLength + ArcLength + DownLength
	if ThermalField.End < zEnd {
		zEnd = ThermalField.End
	}
	for y := Width/YStep - 1; y >= 0; y -= StepY {
		for x := Length/XStep - 1; x >= 0; x -= StepX {
			temperatureData.Down.Down[Width/YStep/2+y/StepY][Length/XStep/2+x/StepX] = ThermalField.Field[ThermalField.End-1][y][x]
			temperatureData.Down.Down[(Width/YStep-1)/2-y/StepY][(Length/XStep-1)/2-x/StepX] = ThermalField.Field[ThermalField.End-1][y][x]
			temperatureData.Down.Down[Width/YStep/2+y/StepY][(Length/XStep-1)/2-x/StepX] = ThermalField.Field[ThermalField.End-1][y][x]
			temperatureData.Down.Down[(Width/YStep-1)/2-y/StepY][Length/XStep/2+x/StepX] = ThermalField.Field[ThermalField.End-1][y][x]
		}
	}
	for y := Width/YStep - 1; y >= 0; y -= StepY {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Down.Left[Width/YStep/2+y/StepY][(x-start)/StepZ] = ThermalField.Field[x][y][Length/XStep-1]
			temperatureData.Down.Left[(Width/YStep-1)/2-y/StepY][(x-start)/StepZ] = ThermalField.Field[x][y][Length/XStep-1]
		}
	}
	for y := Width/YStep - 1; y >= 0; y -= StepY {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Down.Right[Width/YStep/2+y/StepY][(x-start)/StepZ] = ThermalField.Field[x][y][Length/XStep-1]
			temperatureData.Down.Right[(Width/YStep-1)/2-y/StepY][(x-start)/StepZ] = ThermalField.Field[x][y][Length/XStep-1]
		}
	}
	for y := Length/XStep - 1; y >= 0; y -= StepX {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Down.Front[Length/XStep/2+y/StepX][(x-start)/StepZ] = ThermalField.Field[x][Width/YStep-1][y]
			temperatureData.Down.Front[(Length/XStep-1)/2-y/StepX][(x-start)/StepZ] = ThermalField.Field[x][Width/YStep-1][y]
		}
	}
	for y := Length/XStep - 1; y >= 0; y -= StepX {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Down.Back[Length/XStep/2+y/StepX][(x-start)/StepZ] = ThermalField.Field[x][Width/YStep-1][y]
			temperatureData.Down.Back[(Length/XStep-1)/2-y/StepX][(x-start)/StepZ] = ThermalField.Field[x][Width/YStep-1][y]
		}
	}

	//fmt.Printf("up up: 长%d 宽%d")
	//fmt.Println("build data cost: ", time.Since(startTime))
	// temperatureData
}

// 横切面推送数据
func (c *calculatorWithArrDeque) BuildSliceData(index int) *SlicePushDataStruct {
	res := SlicePushDataStruct{Marks: make(map[int]string)}
	slice := make([][]float32, Width/YStep*2)
	for i := 0; i < len(slice); i++ {
		slice[i] = make([]float32, Length/XStep*2)
	}
	originData := c.Field.GetSlice(index)
	// 从右上角的四分之一还原整个二维数组
	for i := 0; i < Width/YStep; i++ {
		for j := 0; j < Length/XStep; j++ {
			slice[i][j] = originData[Width/YStep-1-i][Length/XStep-1-j]
		}
	}
	for i := 0; i < Width/YStep; i++ {
		for j := Length / XStep; j < Length/XStep*2; j++ {
			slice[i][j] = originData[Width/YStep-1-i][j-Length/XStep]
		}
	}
	for i := Width / YStep; i < Width/YStep*2; i++ {
		for j := Length / XStep; j < Length/XStep*2; j++ {
			slice[i][j] = originData[i-Width/YStep][j-Length/XStep]
		}
	}
	for i := Width / YStep; i < Width/YStep*2; i++ {
		for j := 0; j < Length/XStep; j++ {
			slice[i][j] = originData[i-Width/YStep][Length/XStep-1-j]
		}
	}
	res.Slice = slice
	res.Start = c.getFieldStart()
	res.End = ZLength / model.ZStep
	res.Current = c.getFieldEnd()
	res.Marks[0] = "结晶器"
	res.Marks[res.End] = strconv.Itoa(res.End)
	res.Marks[UpLength] = "二冷区"
	return &res
}

// 纵切面推送数据
