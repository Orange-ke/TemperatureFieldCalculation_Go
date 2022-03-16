package calculator

import (
	"lz/model"
	"strconv"
)

var (
	Field = [model.ZLength / model.ZStep]model.ItemType{}

	upUp    = model.ItemType{}
	upLeft  = [model.Width / model.YStep][UpLength / StepZ]float32{}
	upRight = [model.Width / model.YStep][UpLength / StepZ]float32{}
	upFront = [model.Length / model.XStep][UpLength / StepZ]float32{}
	upBack  = [model.Length / model.XStep][UpLength / StepZ]float32{}

	arcLeft  = [model.Width / model.YStep][ArcLength / StepZ]float32{}
	arcRight = [model.Width / model.YStep][ArcLength / StepZ]float32{}
	arcFront = [model.Length / model.XStep][ArcLength / StepZ]float32{}
	arcBack  = [model.Length / model.XStep][ArcLength / StepZ]float32{}

	downDown  = model.ItemType{}
	downLeft  = [model.Width / model.YStep][DownLength / StepZ]float32{}
	downRight = [model.Width / model.YStep][DownLength / StepZ]float32{}
	downFront = [model.Length / model.XStep][DownLength / StepZ]float32{}
	downBack  = [model.Length / model.XStep][DownLength / StepZ]float32{}
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
	Up    *model.ItemType                                        `json:"up"`
	Left  *[model.Width / model.YStep][UpLength / StepZ]float32  `json:"left"`
	Right *[model.Width / model.YStep][UpLength / StepZ]float32  `json:"right"`
	Front *[model.Length / model.XStep][UpLength / StepZ]float32 `json:"front"`
	Back  *[model.Length / model.XStep][UpLength / StepZ]float32 `json:"back"`
}

type ArcSides struct {
	Left  *[model.Width / model.YStep][ArcLength / StepZ]float32  `json:"left"`
	Right *[model.Width / model.YStep][ArcLength / StepZ]float32  `json:"right"`
	Front *[model.Length / model.XStep][ArcLength / StepZ]float32 `json:"front"`
	Back  *[model.Length / model.XStep][ArcLength / StepZ]float32 `json:"back"`
}

type DownSides struct {
	Left  *[model.Width / model.YStep][UpLength / StepZ]float32  `json:"left"`
	Right *[model.Width / model.YStep][UpLength / StepZ]float32  `json:"right"`
	Front *[model.Length / model.XStep][UpLength / StepZ]float32 `json:"front"`
	Back  *[model.Length / model.XStep][UpLength / StepZ]float32 `json:"back"`
	Down  *model.ItemType                                        `json:"down"`
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
	temperatureData := &TemperatureFieldData{
		Up:   upSides,
		Arc:  arcSides,
		Down: downSides,
	}

	ThermalField := &ThermalFieldStruct{
		Field: &Field,
	}

	z := 0
	c.Field.Traverse(func(_ int, item *model.ItemType) {
		Field[z] = *item
		z++
	})

	if c.isFull {
		ThermalField.Start = 0
		ThermalField.End = model.ZLength / model.ZStep
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
		if ThermalField.Start == model.ZLength/model.ZStep {
			return
		}
	}
	//startTime := time.Now()
	// up
	for y := model.Width/model.YStep - 1; y >= 0; y -= StepY {
		for x := model.Length/model.XStep - 1; x >= 0; x -= StepX {
			temperatureData.Up.Up[model.Width/model.YStep/2+y/StepY][model.Length/model.XStep/2+x/StepX] = ThermalField.Field[ThermalField.Start][y][x]
			temperatureData.Up.Up[model.Width/model.YStep/2-1-y/StepY][model.Length/model.XStep/2-1-x/StepX] = ThermalField.Field[ThermalField.Start][y][x]
			temperatureData.Up.Up[model.Width/model.YStep/2+y/StepY][model.Length/model.XStep/2-1-x/StepX] = ThermalField.Field[ThermalField.Start][y][x]
			temperatureData.Up.Up[model.Width/model.YStep/2-1-y/StepY][model.Length/model.XStep/2+x/StepX] = ThermalField.Field[ThermalField.Start][y][x]
		}
	}
	start := 0
	zStart := ThermalField.Start
	zEnd := UpLength
	if ThermalField.End < zEnd {
		zEnd = ThermalField.End
	}
	for y := model.Width/model.YStep - 1; y >= 0; y -= StepY {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Up.Left[model.Width/model.YStep/2+y/StepY][x/StepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
			temperatureData.Up.Left[model.Width/model.YStep/2-1-y/StepY][x/StepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
		}
	}
	for y := model.Width/model.YStep - 1; y >= 0; y -= StepY {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Up.Right[model.Width/model.YStep/2+y/StepY][x/StepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
			temperatureData.Up.Right[model.Width/model.YStep/2-1-y/StepY][x/StepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
		}
	}
	for y := model.Length/model.XStep - 1; y >= 0; y -= StepX {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Up.Front[model.Length/model.XStep/2+y/StepX][x/StepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
			temperatureData.Up.Front[model.Length/model.XStep/2-y/StepX-1][x/StepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
		}
	}
	for y := model.Length/model.XStep - 1; y >= 0; y -= StepX {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Up.Back[model.Length/model.XStep/2+y/StepX][x/StepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
			temperatureData.Up.Back[model.Length/model.XStep/2-y/StepX-1][x/StepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
		}
	}

	start = UpLength
	zStart = max(UpLength, ThermalField.Start)
	zEnd = UpLength + ArcLength
	if ThermalField.End < zEnd {
		zEnd = ThermalField.End
	}
	for y := model.Width/model.YStep - 1; y >= 0; y -= StepY {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Arc.Left[model.Width/model.YStep/2+y/StepY][(x-start)/StepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
			temperatureData.Arc.Left[model.Width/model.YStep/2-1-y/StepY][(x-start)/StepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
		}
	}
	for y := model.Width/model.YStep - 1; y >= 0; y -= StepY {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Arc.Right[model.Width/model.YStep/2+y/StepY][(x-start)/StepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
			temperatureData.Arc.Right[model.Width/model.YStep/2-1-y/StepY][(x-start)/StepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
		}
	}
	for y := model.Length/model.XStep - 1; y >= 0; y -= StepX {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Arc.Front[model.Length/model.XStep/2+y/StepX][(x-start)/StepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
			temperatureData.Arc.Front[model.Length/model.XStep/2-y/StepX-1][(x-start)/StepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
		}
	}
	for y := model.Length/model.XStep - 1; y >= 0; y -= StepX {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Arc.Back[model.Length/model.XStep/2+y/StepX][(x-start)/StepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
			temperatureData.Arc.Back[model.Length/model.XStep/2-y/StepX-1][(x-start)/StepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
		}
	}

	start = UpLength + ArcLength
	zStart = max(UpLength+ArcLength, ThermalField.Start)
	zEnd = UpLength + ArcLength + DownLength
	if ThermalField.End < zEnd {
		zEnd = ThermalField.End
	}
	for y := model.Width/model.YStep - 1; y >= 0; y -= StepY {
		for x := model.Length/model.XStep - 1; x >= 0; x -= StepX {
			temperatureData.Down.Down[model.Width/model.YStep/2+y/StepY][model.Length/model.XStep/2+x/StepX] = ThermalField.Field[ThermalField.End-1][y][x]
			temperatureData.Down.Down[model.Width/model.YStep/2-1-y/StepY][model.Length/model.XStep/2-1-x/StepX] = ThermalField.Field[ThermalField.End-1][y][x]
			temperatureData.Down.Down[model.Width/model.YStep/2+y/StepY][model.Length/model.XStep/2-1-x/StepX] = ThermalField.Field[ThermalField.End-1][y][x]
			temperatureData.Down.Down[model.Width/model.YStep/2-1-y/StepY][model.Length/model.XStep/2+x/StepX] = ThermalField.Field[ThermalField.End-1][y][x]
		}
	}
	for y := model.Width/model.YStep - 1; y >= 0; y -= StepY {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Down.Left[model.Width/model.YStep/2+y/StepY][(x-start)/StepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
			temperatureData.Down.Left[model.Width/model.YStep/2-y/StepY-1][(x-start)/StepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
		}
	}
	for y := model.Width/model.YStep - 1; y >= 0; y -= StepY {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Down.Right[model.Width/model.YStep/2+y/StepY][(x-start)/StepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
			temperatureData.Down.Right[model.Width/model.YStep/2-y/StepY-1][(x-start)/StepZ] = ThermalField.Field[x][y][model.Length/model.XStep-1]
		}
	}
	for y := model.Length/model.XStep - 1; y >= 0; y -= StepX {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Down.Front[model.Length/model.XStep/2+y/StepX][(x-start)/StepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
			temperatureData.Down.Front[model.Length/model.XStep/2-y/StepX-1][(x-start)/StepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
		}
	}
	for y := model.Length/model.XStep - 1; y >= 0; y -= StepX {
		for x := zEnd - 1; x >= zStart; x -= StepZ {
			temperatureData.Down.Back[model.Length/model.XStep/2+y/StepX][(x-start)/StepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
			temperatureData.Down.Back[model.Length/model.XStep/2-y/StepX-1][(x-start)/StepZ] = ThermalField.Field[x][model.Width/model.YStep-1][y]
		}
	}

	//fmt.Printf("up up: 长%d 宽%d")
	//fmt.Println("build data cost: ", time.Since(startTime))
	// temperatureData
}

// 横切面推送数据
func (c *calculatorWithArrDeque) BuildSliceData(index int) *SlicePushDataStruct {
	res := SlicePushDataStruct{Marks: make(map[int]string)}
	slice := [model.Width / model.YStep * 2][model.Length / model.XStep * 2]float32{}
	originData := c.Field.GetSlice(index)
	// 从右上角的四分之一还原整个二维数组
	for i := 0; i < model.Width/model.YStep; i++ {
		for j := 0; j < model.Length/model.XStep; j++ {
			slice[i][j] = originData[model.Width/model.YStep-1-i][model.Length/model.XStep-1-j]
		}
	}
	for i := 0; i < model.Width/model.YStep; i++ {
		for j := model.Length / model.XStep; j < model.Length/model.XStep*2; j++ {
			slice[i][j] = originData[model.Width/model.YStep-1-i][j-model.Length/model.XStep]
		}
	}
	for i := model.Width / model.YStep; i < model.Width/model.YStep*2; i++ {
		for j := model.Length / model.XStep; j < model.Length/model.XStep*2; j++ {
			slice[i][j] = originData[i-model.Width/model.YStep][j-model.Length/model.XStep]
		}
	}
	for i := model.Width / model.YStep; i < model.Width/model.YStep*2; i++ {
		for j := 0; j < model.Length/model.XStep; j++ {
			slice[i][j] = originData[i-model.Width/model.YStep][model.Length/model.XStep-1-j]
		}
	}
	res.Slice = &slice
	res.Start = c.getFieldStart()
	res.End = model.ZLength / model.ZStep
	res.Current = c.getFieldEnd()
	res.Marks[0] = "结晶器"
	res.Marks[res.End] = strconv.Itoa(res.End)
	res.Marks[UpLength] = "二冷区"
	return &res
}

// 纵切面推送数据
