package calculator

import (
	"fmt"
	"lz/model"
)

type SlicePushDataStruct struct {
	Slice   [][]float32 `json:"slice"`
	Start   int         `json:"start"`
	End     int         `json:"end"`
	Current int         `json:"current"`
}

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

	StepX = 2
	StepY = 1
	StepZ = 2

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
	temperatureData := &TemperatureFieldData{
		Sides: sides,
	}

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
type SliceInfo struct {
	HorizontalSolidThickness  int         `json:"horizontal_solid_thickness"`
	VerticalSolidThickness    int         `json:"vertical_solid_thickness"`
	HorizontalLiquidThickness int         `json:"horizontal_liquid_thickness"`
	VerticalLiquidThickness   int         `json:"vertical_liquid_thickness"`
	Slice                     [][]float32 `json:"slice"`
}

func (c *calculatorWithArrDeque) GenerateSLiceInfo(index int) *SliceInfo {
	return c.buildSliceGenerateData(index)
}

func (c *calculatorWithArrDeque) buildSliceGenerateData(index int) *SliceInfo {
	solidTemp := c.steel1.SolidPhaseTemperature
	liquidTemp := c.steel1.LiquidPhaseTemperature
	sliceInfo := &SliceInfo{}
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
	sliceInfo.Slice = slice
	length := Length/XStep - 1
	width := Width/YStep - 1
	for i := length; i >= 0; i-- {
		if originData[0][i] <= solidTemp {
			sliceInfo.HorizontalSolidThickness = XStep * (length - i + 1)
		}
	}
	for i := length; i >= 0; i-- {
		if originData[0][i] <= liquidTemp {
			sliceInfo.HorizontalLiquidThickness = XStep * (length - i + 1)
		}
	}

	for j := width; j >= 0; j-- {
		if originData[j][0] <= solidTemp {
			sliceInfo.VerticalSolidThickness = YStep * (width - j + 1)
		}
	}
	for j := width; j >= 0; j-- {
		if originData[j][0] <= liquidTemp {
			sliceInfo.VerticalLiquidThickness = YStep * (width - j + 1)
		}
	}
	return sliceInfo
}

type VerticalSliceData1 struct {
	CenterOuter [][2]float32 `json:"center_outer"`
	CenterInner [][2]float32 `json:"center_inner"`

	EdgeOuter [][2]float32 `json:"edge_outer"`
	EdgeInner [][2]float32 `json:"edge_inner"`
}

func (c *calculatorWithArrDeque) GenerateVerticalSlice1Data() *VerticalSliceData1 {
	var index int
	res := &VerticalSliceData1{
		CenterOuter: make([][2]float32, 0),
		CenterInner: make([][2]float32, 0),
		EdgeOuter:   make([][2]float32, 0),
		EdgeInner:   make([][2]float32, 0),
	}
	step := 0
	c.Field.Traverse(func(z int, item *model.ItemType) {
		step++
		if step == 5 {
			index = Length/XStep - 1
			res.CenterOuter = append(res.CenterOuter, [2]float32{float32((z + 1) * model.ZStep), item[Width/YStep-1][Length/XStep-1-index]})
			res.CenterInner = append(res.CenterInner, [2]float32{float32((z + 1) * model.ZStep), item[0][Length/XStep-1-index]})

			index = 0
			res.EdgeOuter = append(res.EdgeOuter, [2]float32{float32((z + 1) * model.ZStep), item[Width/YStep-1][Length/XStep-1-index]})
			res.EdgeInner = append(res.EdgeInner, [2]float32{float32((z + 1) * model.ZStep), item[0][Length/XStep-1-index]})

			step = 0
		}
	}, 0, c.Field.Size())
	return res
}

type VerticalSliceData2 struct {
	Length        int         `json:"length"`
	VerticalSlice [][]float32 `json:"vertical_slice"`
	Solid         []int       `json:"solid"`
	Liquid        []int       `json:"liquid"`
	SolidJoin     Join        `json:"solid_join"`
	LiquidJoin    Join        `json:"liquid_join"`
}

type Join struct {
	IsJoin    bool `json:"is_join"`
	JoinIndex int  `json:"join_index"`
}

func (c *calculatorWithArrDeque) GenerateVerticalSlice2Data(reqData model.VerticalReqData) *VerticalSliceData2 {
	solidTemp := c.steel1.SolidPhaseTemperature
	liquidTemp := c.steel1.LiquidPhaseTemperature
	index := reqData.Index
	zScale := reqData.ZScale
	res := &VerticalSliceData2{
		Length:        c.Field.Size(),
		VerticalSlice: make([][]float32, c.Field.Size()/zScale),
		Solid:         make([]int, c.Field.Size()),
		Liquid:        make([]int, c.Field.Size()),
	}

	for i := 0; i < len(res.VerticalSlice); i++ {
		res.VerticalSlice[i] = make([]float32, Width/YStep*2)
	}

	var temp float32
	var solidJoinSet, liquidJoinSet bool
	step := 0
	zIndex := 0
	c.Field.Traverse(func(z int, item *model.ItemType) {
		step++
		if step == zScale {
			for i := 0; i < Width/YStep; i++ {
				res.VerticalSlice[zIndex][Width/YStep+i] = item[i][Length/XStep-1-index]
			}
			for i := Width/YStep - 1; i >= 0; i-- {
				res.VerticalSlice[zIndex][Width/YStep-1-i] = item[i][Length/XStep-1-index]
			}
			step = 0
			zIndex++
		}
		for i := 0; i < Width/YStep; i++ {
			temp = item[i][Length/XStep-1-index]
			if temp <= solidTemp {
				res.Solid[z] = Width/YStep - i
				if res.Solid[z] == Width/YStep && !solidJoinSet {
					res.SolidJoin.IsJoin = true
					res.SolidJoin.JoinIndex = z
					solidJoinSet = true
					fmt.Println(res.SolidJoin)
				}
				break
			} else {
				res.Solid[z] = 0
			}
		}

		for i := 0; i < Width/YStep; i++ {
			temp = item[i][Length/XStep-1-index]
			if temp <= liquidTemp {
				res.Liquid[z] = Width/YStep - i
				if res.Liquid[z] == Width/YStep && !liquidJoinSet {
					res.LiquidJoin.IsJoin = true
					res.LiquidJoin.JoinIndex = z
					liquidJoinSet = true
					fmt.Println(res.LiquidJoin)
				}
				break
			} else {
				res.Liquid[z] = 0
			}
		}
	}, 0, c.Field.Size())
	fmt.Println(res)
	return res
}
