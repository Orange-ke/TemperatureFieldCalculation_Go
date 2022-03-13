package calculator

import (
	"lz/casting_machine"
	"strconv"
)

var (
	Field = [casting_machine.ZLength / casting_machine.ZStep]casting_machine.ItemType{}

	upUp    = casting_machine.ItemType{}
	upLeft  = [casting_machine.Width / casting_machine.YStep][casting_machine.UpLength / casting_machine.StepZ]float32{}
	upRight = [casting_machine.Width / casting_machine.YStep][casting_machine.UpLength / casting_machine.StepZ]float32{}
	upFront = [casting_machine.Length / casting_machine.XStep][casting_machine.UpLength / casting_machine.StepZ]float32{}
	upBack  = [casting_machine.Length / casting_machine.XStep][casting_machine.UpLength / casting_machine.StepZ]float32{}

	arcLeft  = [casting_machine.Width / casting_machine.YStep][casting_machine.ArcLength / casting_machine.StepZ]float32{}
	arcRight = [casting_machine.Width / casting_machine.YStep][casting_machine.ArcLength / casting_machine.StepZ]float32{}
	arcFront = [casting_machine.Length / casting_machine.XStep][casting_machine.ArcLength / casting_machine.StepZ]float32{}
	arcBack  = [casting_machine.Length / casting_machine.XStep][casting_machine.ArcLength / casting_machine.StepZ]float32{}

	downDown  = casting_machine.ItemType{}
	downLeft  = [casting_machine.Width / casting_machine.YStep][casting_machine.DownLength / casting_machine.StepZ]float32{}
	downRight = [casting_machine.Width / casting_machine.YStep][casting_machine.DownLength / casting_machine.StepZ]float32{}
	downFront = [casting_machine.Length / casting_machine.XStep][casting_machine.DownLength / casting_machine.StepZ]float32{}
	downBack  = [casting_machine.Length / casting_machine.XStep][casting_machine.DownLength / casting_machine.StepZ]float32{}
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
	Up    *casting_machine.ItemType                                        `json:"up"`
	Left  *[casting_machine.Width / casting_machine.YStep][casting_machine.UpLength / casting_machine.StepZ]float32  `json:"left"`
	Right *[casting_machine.Width / casting_machine.YStep][casting_machine.UpLength / casting_machine.StepZ]float32  `json:"right"`
	Front *[casting_machine.Length / casting_machine.XStep][casting_machine.UpLength / casting_machine.StepZ]float32 `json:"front"`
	Back  *[casting_machine.Length / casting_machine.XStep][casting_machine.UpLength / casting_machine.StepZ]float32 `json:"back"`
}

type ArcSides struct {
	Left  *[casting_machine.Width / casting_machine.YStep][casting_machine.ArcLength / casting_machine.StepZ]float32  `json:"left"`
	Right *[casting_machine.Width / casting_machine.YStep][casting_machine.ArcLength / casting_machine.StepZ]float32  `json:"right"`
	Front *[casting_machine.Length / casting_machine.XStep][casting_machine.ArcLength / casting_machine.StepZ]float32 `json:"front"`
	Back  *[casting_machine.Length / casting_machine.XStep][casting_machine.ArcLength / casting_machine.StepZ]float32 `json:"back"`
}

type DownSides struct {
	Left  *[casting_machine.Width / casting_machine.YStep][casting_machine.UpLength / casting_machine.StepZ]float32  `json:"left"`
	Right *[casting_machine.Width / casting_machine.YStep][casting_machine.UpLength / casting_machine.StepZ]float32  `json:"right"`
	Front *[casting_machine.Length / casting_machine.XStep][casting_machine.UpLength / casting_machine.StepZ]float32 `json:"front"`
	Back  *[casting_machine.Length / casting_machine.XStep][casting_machine.UpLength / casting_machine.StepZ]float32 `json:"back"`
	Down  *casting_machine.ItemType                                        `json:"down"`
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
	c.Field.Traverse(func(_ int, item *casting_machine.ItemType) {
		Field[z] = *item
		z++
	})

	if c.isFull {
		ThermalField.Start = 0
		ThermalField.End = casting_machine.ZLength / casting_machine.ZStep
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
		if ThermalField.Start == casting_machine.ZLength / casting_machine.ZStep {
			return
		}
	}
	//startTime := time.Now()
	// up
	for y := casting_machine.Width/casting_machine.YStep - 1; y >= 0; y -= casting_machine.StepY {
		for x := casting_machine.Length/casting_machine.XStep - 1; x >= 0; x -= casting_machine.StepX {
			temperatureData.Up.Up[casting_machine.Width/casting_machine.YStep/2+y/casting_machine.StepY][casting_machine.Length/casting_machine.XStep/2+x/casting_machine.StepX] = ThermalField.Field[ThermalField.Start][y][x]
			temperatureData.Up.Up[casting_machine.Width/casting_machine.YStep/2-1-y/casting_machine.StepY][casting_machine.Length/casting_machine.XStep/2-1-x/casting_machine.StepX] = ThermalField.Field[ThermalField.Start][y][x]
			temperatureData.Up.Up[casting_machine.Width/casting_machine.YStep/2+y/casting_machine.StepY][casting_machine.Length/casting_machine.XStep/2-1-x/casting_machine.StepX] = ThermalField.Field[ThermalField.Start][y][x]
			temperatureData.Up.Up[casting_machine.Width/casting_machine.YStep/2-1-y/casting_machine.StepY][casting_machine.Length/casting_machine.XStep/2+x/casting_machine.StepX] = ThermalField.Field[ThermalField.Start][y][x]
		}
	}
	start := 0
	zStart := ThermalField.Start
	zEnd := casting_machine.UpLength
	if ThermalField.End < zEnd {
		zEnd = ThermalField.End
	}
	for y := casting_machine.Width/casting_machine.YStep - 1; y >= 0; y -= casting_machine.StepY {
		for x := zEnd - 1; x >= zStart; x -= casting_machine.StepZ {
			temperatureData.Up.Left[casting_machine.Width/casting_machine.YStep/2+y/casting_machine.StepY][x/casting_machine.StepZ] = ThermalField.Field[x][y][casting_machine.Length/casting_machine.XStep-1]
			temperatureData.Up.Left[casting_machine.Width/casting_machine.YStep/2-1-y/casting_machine.StepY][x/casting_machine.StepZ] = ThermalField.Field[x][y][casting_machine.Length/casting_machine.XStep-1]
		}
	}
	for y := casting_machine.Width/casting_machine.YStep - 1; y >= 0; y -= casting_machine.StepY {
		for x := zEnd - 1; x >= zStart; x -= casting_machine.StepZ {
			temperatureData.Up.Right[casting_machine.Width/casting_machine.YStep/2+y/casting_machine.StepY][x/casting_machine.StepZ] = ThermalField.Field[x][y][casting_machine.Length/casting_machine.XStep-1]
			temperatureData.Up.Right[casting_machine.Width/casting_machine.YStep/2-1-y/casting_machine.StepY][x/casting_machine.StepZ] = ThermalField.Field[x][y][casting_machine.Length/casting_machine.XStep-1]
		}
	}
	for y := casting_machine.Length/casting_machine.XStep - 1; y >= 0; y -= casting_machine.StepX {
		for x := zEnd - 1; x >= zStart; x -= casting_machine.StepZ {
			temperatureData.Up.Front[casting_machine.Length/casting_machine.XStep/2+y/casting_machine.StepX][x/casting_machine.StepZ] = ThermalField.Field[x][casting_machine.Width/casting_machine.YStep-1][y]
			temperatureData.Up.Front[casting_machine.Length/casting_machine.XStep/2-y/casting_machine.StepX-1][x/casting_machine.StepZ] = ThermalField.Field[x][casting_machine.Width/casting_machine.YStep-1][y]
		}
	}
	for y := casting_machine.Length/casting_machine.XStep - 1; y >= 0; y -= casting_machine.StepX {
		for x := zEnd - 1; x >= zStart; x -= casting_machine.StepZ {
			temperatureData.Up.Back[casting_machine.Length/casting_machine.XStep/2+y/casting_machine.StepX][x/casting_machine.StepZ] = ThermalField.Field[x][casting_machine.Width/casting_machine.YStep-1][y]
			temperatureData.Up.Back[casting_machine.Length/casting_machine.XStep/2-y/casting_machine.StepX-1][x/casting_machine.StepZ] = ThermalField.Field[x][casting_machine.Width/casting_machine.YStep-1][y]
		}
	}

	start = casting_machine.UpLength
	zStart = max(casting_machine.UpLength, ThermalField.Start)
	zEnd = casting_machine.UpLength + casting_machine.ArcLength
	if ThermalField.End < zEnd {
		zEnd = ThermalField.End
	}
	for y := casting_machine.Width/casting_machine.YStep - 1; y >= 0; y -= casting_machine.StepY {
		for x := zEnd - 1; x >= zStart; x -= casting_machine.StepZ {
			temperatureData.Arc.Left[casting_machine.Width/casting_machine.YStep/2+y/casting_machine.StepY][(x-start)/casting_machine.StepZ] = ThermalField.Field[x][y][casting_machine.Length/casting_machine.XStep-1]
			temperatureData.Arc.Left[casting_machine.Width/casting_machine.YStep/2-1-y/casting_machine.StepY][(x-start)/casting_machine.StepZ] = ThermalField.Field[x][y][casting_machine.Length/casting_machine.XStep-1]
		}
	}
	for y := casting_machine.Width/casting_machine.YStep - 1; y >= 0; y -= casting_machine.StepY {
		for x := zEnd - 1; x >= zStart; x -= casting_machine.StepZ {
			temperatureData.Arc.Right[casting_machine.Width/casting_machine.YStep/2+y/casting_machine.StepY][(x-start)/casting_machine.StepZ] = ThermalField.Field[x][y][casting_machine.Length/casting_machine.XStep-1]
			temperatureData.Arc.Right[casting_machine.Width/casting_machine.YStep/2-1-y/casting_machine.StepY][(x-start)/casting_machine.StepZ] = ThermalField.Field[x][y][casting_machine.Length/casting_machine.XStep-1]
		}
	}
	for y := casting_machine.Length/casting_machine.XStep - 1; y >= 0; y -= casting_machine.StepX {
		for x := zEnd - 1; x >= zStart; x -= casting_machine.StepZ {
			temperatureData.Arc.Front[casting_machine.Length/casting_machine.XStep/2+y/casting_machine.StepX][(x-start)/casting_machine.StepZ] = ThermalField.Field[x][casting_machine.Width/casting_machine.YStep-1][y]
			temperatureData.Arc.Front[casting_machine.Length/casting_machine.XStep/2-y/casting_machine.StepX-1][(x-start)/casting_machine.StepZ] = ThermalField.Field[x][casting_machine.Width/casting_machine.YStep-1][y]
		}
	}
	for y := casting_machine.Length/casting_machine.XStep - 1; y >= 0; y -= casting_machine.StepX {
		for x := zEnd - 1; x >= zStart; x -= casting_machine.StepZ {
			temperatureData.Arc.Back[casting_machine.Length/casting_machine.XStep/2+y/casting_machine.StepX][(x-start)/casting_machine.StepZ] = ThermalField.Field[x][casting_machine.Width/casting_machine.YStep-1][y]
			temperatureData.Arc.Back[casting_machine.Length/casting_machine.XStep/2-y/casting_machine.StepX-1][(x-start)/casting_machine.StepZ] = ThermalField.Field[x][casting_machine.Width/casting_machine.YStep-1][y]
		}
	}

	start = casting_machine.UpLength + casting_machine.ArcLength
	zStart = max(casting_machine.UpLength + casting_machine.ArcLength, ThermalField.Start)
	zEnd = casting_machine.UpLength + casting_machine.ArcLength + casting_machine.DownLength
	if ThermalField.End < zEnd {
		zEnd = ThermalField.End
	}
	for y := casting_machine.Width/casting_machine.YStep - 1; y >= 0; y -= casting_machine.StepY {
		for x := casting_machine.Length/casting_machine.XStep - 1; x >= 0; x -= casting_machine.StepX {
			temperatureData.Down.Down[casting_machine.Width/casting_machine.YStep/2+y/casting_machine.StepY][casting_machine.Length/casting_machine.XStep/2+x/casting_machine.StepX] = ThermalField.Field[ThermalField.End-1][y][x]
			temperatureData.Down.Down[casting_machine.Width/casting_machine.YStep/2-1-y/casting_machine.StepY][casting_machine.Length/casting_machine.XStep/2-1-x/casting_machine.StepX] = ThermalField.Field[ThermalField.End-1][y][x]
			temperatureData.Down.Down[casting_machine.Width/casting_machine.YStep/2+y/casting_machine.StepY][casting_machine.Length/casting_machine.XStep/2-1-x/casting_machine.StepX] = ThermalField.Field[ThermalField.End-1][y][x]
			temperatureData.Down.Down[casting_machine.Width/casting_machine.YStep/2-1-y/casting_machine.StepY][casting_machine.Length/casting_machine.XStep/2+x/casting_machine.StepX] = ThermalField.Field[ThermalField.End-1][y][x]
		}
	}
	for y := casting_machine.Width/casting_machine.YStep - 1; y >= 0; y -= casting_machine.StepY {
		for x := zEnd - 1; x >= zStart; x -= casting_machine.StepZ {
			temperatureData.Down.Left[casting_machine.Width/casting_machine.YStep/2+y/casting_machine.StepY][(x-start)/casting_machine.StepZ] = ThermalField.Field[x][y][casting_machine.Length/casting_machine.XStep-1]
			temperatureData.Down.Left[casting_machine.Width/casting_machine.YStep/2-y/casting_machine.StepY-1][(x-start)/casting_machine.StepZ] = ThermalField.Field[x][y][casting_machine.Length/casting_machine.XStep-1]
		}
	}
	for y := casting_machine.Width/casting_machine.YStep - 1; y >= 0; y -= casting_machine.StepY {
		for x := zEnd - 1; x >= zStart; x -= casting_machine.StepZ {
			temperatureData.Down.Right[casting_machine.Width/casting_machine.YStep/2+y/casting_machine.StepY][(x-start)/casting_machine.StepZ] = ThermalField.Field[x][y][casting_machine.Length/casting_machine.XStep-1]
			temperatureData.Down.Right[casting_machine.Width/casting_machine.YStep/2-y/casting_machine.StepY-1][(x-start)/casting_machine.StepZ] = ThermalField.Field[x][y][casting_machine.Length/casting_machine.XStep-1]
		}
	}
	for y := casting_machine.Length/casting_machine.XStep - 1; y >= 0; y -= casting_machine.StepX {
		for x := zEnd - 1; x >= zStart; x -= casting_machine.StepZ {
			temperatureData.Down.Front[casting_machine.Length/casting_machine.XStep/2+y/casting_machine.StepX][(x-start)/casting_machine.StepZ] = ThermalField.Field[x][casting_machine.Width/casting_machine.YStep-1][y]
			temperatureData.Down.Front[casting_machine.Length/casting_machine.XStep/2-y/casting_machine.StepX-1][(x-start)/casting_machine.StepZ] = ThermalField.Field[x][casting_machine.Width/casting_machine.YStep-1][y]
		}
	}
	for y := casting_machine.Length/casting_machine.XStep - 1; y >= 0; y -= casting_machine.StepX {
		for x := zEnd - 1; x >= zStart; x -= casting_machine.StepZ {
			temperatureData.Down.Back[casting_machine.Length/casting_machine.XStep/2+y/casting_machine.StepX][(x-start)/casting_machine.StepZ] = ThermalField.Field[x][casting_machine.Width/casting_machine.YStep-1][y]
			temperatureData.Down.Back[casting_machine.Length/casting_machine.XStep/2-y/casting_machine.StepX-1][(x-start)/casting_machine.StepZ] = ThermalField.Field[x][casting_machine.Width/casting_machine.YStep-1][y]
		}
	}

	//fmt.Printf("up up: 长%d 宽%d")
	//fmt.Println("build data cost: ", time.Since(startTime))
	// temperatureData
}

// 横切面推送数据
func (c *calculatorWithArrDeque) BuildSliceData(index int) *SlicePushDataStruct {
	res := SlicePushDataStruct{Marks: make(map[int]string)}
	slice := [casting_machine.Width / casting_machine.YStep * 2][casting_machine.Length / casting_machine.XStep * 2]float32{}
	originData := c.Field.GetSlice(index)
	// 从右上角的四分之一还原整个二维数组
	for i := 0; i < casting_machine.Width/casting_machine.YStep; i++ {
		for j := 0; j < casting_machine.Length/casting_machine.XStep; j++ {
			slice[i][j] = originData[casting_machine.Width/casting_machine.YStep-1-i][casting_machine.Length/casting_machine.XStep-1-j]
		}
	}
	for i := 0; i < casting_machine.Width/casting_machine.YStep; i++ {
		for j := casting_machine.Length / casting_machine.XStep; j < casting_machine.Length/casting_machine.XStep*2; j++ {
			slice[i][j] = originData[casting_machine.Width/casting_machine.YStep-1-i][j-casting_machine.Length/casting_machine.XStep]
		}
	}
	for i := casting_machine.Width / casting_machine.YStep; i < casting_machine.Width/casting_machine.YStep*2; i++ {
		for j := casting_machine.Length / casting_machine.XStep; j < casting_machine.Length/casting_machine.XStep*2; j++ {
			slice[i][j] = originData[i-casting_machine.Width/casting_machine.YStep][j-casting_machine.Length/casting_machine.XStep]
		}
	}
	for i := casting_machine.Width / casting_machine.YStep; i < casting_machine.Width/casting_machine.YStep*2; i++ {
		for j := 0; j < casting_machine.Length/casting_machine.XStep; j++ {
			slice[i][j] = originData[i-casting_machine.Width/casting_machine.YStep][casting_machine.Length/casting_machine.XStep-1-j]
		}
	}
	res.Slice = &slice
	res.Start = c.getFieldStart()
	res.End = casting_machine.ZLength / casting_machine.ZStep
	res.Current = c.getFieldEnd()
	res.Marks[0] = "结晶器"
	res.Marks[res.End] = strconv.Itoa(res.End)
	res.Marks[casting_machine.UpLength] = "二冷区"
	return &res
}

// 总切面推送数据
