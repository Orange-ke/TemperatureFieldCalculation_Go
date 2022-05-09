package calculator

import (
	"fmt"
	"lz/model"
)

type TemperatureFieldData struct {
	XScale int    `json:"x_scale"`
	YScale int    `json:"y_scale"`
	ZScale int    `json:"z_scale"`
	Start  int    `json:"start"`   // 切片开始位置
	End    int    `json:"end"`     // 切片结束位置
	IsFull bool   `json:"is_full"` // 切片是否充满铸机
	IsTail bool   `json:"is_tail"` // 是否拉尾坯
	Sides  *Sides `json:"sides"`
}

type Sides struct {
	Up    [][]float32 `json:"up"`
	Left  [][]float32 `json:"left"`
	Right [][]float32 `json:"right"`
	Front [][]float32 `json:"front"`
	Back  [][]float32 `json:"back"`
	Down  [][]float32 `json:"down"`
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

var (
	UpLength   float32
	ArcLength  float32
	DownLength float32

	StepX = 1
	StepY = 1
	StepZ = 1

	width  int
	length int

	sides = &Sides{}
)

func initPushData(up, arc, down float32) {
	UpLength, ArcLength, DownLength = up, arc, down
	width = Width / YStep / StepY * 2
	length = Length / XStep / StepX * 2
	fmt.Println("pushData:", width, length, ZLength/ZStep/StepZ)
	sides = &Sides{
		Up:    make([][]float32, width),
		Left:  make([][]float32, ZLength/ZStep/StepZ),
		Right: make([][]float32, ZLength/ZStep/StepZ),
		Front: make([][]float32, ZLength/ZStep/StepZ),
		Back:  make([][]float32, ZLength/ZStep/StepZ),
		Down:  make([][]float32, width),
	}

	for i := 0; i < width; i++ {
		sides.Up[i] = make([]float32, length)
		sides.Down[i] = make([]float32, length)
	}
	for i := 0; i < ZLength/ZStep/StepZ; i++ {
		sides.Left[i] = make([]float32, width)
		sides.Right[i] = make([]float32, width)
		sides.Front[i] = make([]float32, length)
		sides.Back[i] = make([]float32, length)
	}
}

func (c *calculatorWithArrDeque) BuildData() *TemperatureFieldData {
	fmt.Println("buildData", c.Field.Size())
	sides.Left = sides.Left[:c.Field.Size()]
	sides.Right = sides.Right[:c.Field.Size()]
	sides.Front = sides.Front[:c.Field.Size()]
	sides.Back = sides.Back[:c.Field.Size()]
	temperatureData := &TemperatureFieldData{
		Sides: sides,
	}
	//for i := 0; i < 2000; i++ {
	//	c.Field.RemoveLast()
	//}
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

	//startTime := time.Now()
	startSlice := c.Field.GetSlice(0)
	EndSlice := c.Field.GetSlice(c.Field.Size() - 1)
	for y := Width/YStep - 1; y >= 0; y -= StepY {
		for x := Length/XStep - 1; x >= 0; x -= StepX {
			temperatureData.Sides.Up[width/2+y/StepY][length/2+x/StepX] = startSlice[y][x]
			temperatureData.Sides.Up[(width/2-1)-y/StepY][(length/2-1)-x/StepX] = startSlice[y][x]
			temperatureData.Sides.Up[width/2+y/StepY][(length/2-1)-x/StepX] = startSlice[y][x]
			temperatureData.Sides.Up[(width/2-1)-y/StepY][length/2+x/StepX] = startSlice[y][x]
		}
	}

	for z := c.Field.Size() - 1; z >= 0; z -= StepZ {
		slice := c.Field.GetSlice(z)
		for x := Length/XStep - 1; x >= 0; x -= StepX {
			temperatureData.Sides.Front[z/StepZ][length/2+x/StepX] = slice[Width/YStep-1][x]
			temperatureData.Sides.Front[z/StepZ][length/2-1-x/StepX] = slice[Width/YStep-1][x]

			temperatureData.Sides.Back[z/StepZ][length/2+x/StepX] = slice[Width/YStep-1][x]
			temperatureData.Sides.Back[z/StepZ][length/2-1-x/StepX] = slice[Width/YStep-1][x]
		}

		for y := Width/YStep - 1; y >= 0; y -= StepY {
			temperatureData.Sides.Left[z/StepZ][width/2+y/StepY] = slice[y][Length/XStep-1]
			temperatureData.Sides.Left[z/StepZ][width/2-1-y/StepY] = slice[y][Length/XStep-1]

			temperatureData.Sides.Right[z/StepZ][width/2+y/StepY] = slice[y][Length/XStep-1]
			temperatureData.Sides.Right[z/StepZ][width/2-1-y/StepY] = slice[y][Length/XStep-1]
		}
	}

	for y := Width/YStep - 1; y >= 0; y -= StepY {
		for x := Length/XStep - 1; x >= 0; x -= StepX {
			temperatureData.Sides.Down[width/2+y/StepY][length/2+x/StepX] = EndSlice[y][x]
			temperatureData.Sides.Down[(width/2-1)-y/StepY][(length/2-1)-x/StepX] = EndSlice[y][x]
			temperatureData.Sides.Down[width/2+y/StepY][(length/2-1)-x/StepX] = EndSlice[y][x]
			temperatureData.Sides.Down[(width/2-1)-y/StepY][length/2+x/StepX] = EndSlice[y][x]
		}
	}

	temperatureData.XScale = StepX
	temperatureData.YScale = StepY
	temperatureData.ZScale = StepZ
	temperatureData.Start = c.start
	temperatureData.End = c.end
	temperatureData.IsFull = c.Field.IsFull()
	temperatureData.IsTail = c.isTail
	// fmt.Println("build data cost: ", time.Since(startTime))
	return temperatureData
}

// 横切面推送数据
func (c *calculatorWithArrDeque) BuildSliceData(index int) *SlicePushDataStruct {
	res := SlicePushDataStruct{}
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
	return &res
}

// 纵切面推送数据
