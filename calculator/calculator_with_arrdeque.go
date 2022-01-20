package calculator

import (
	"fmt"
	"lz/deque"
	"lz/model"
	"sync"
	"time"
)

type calculatorWithArrDeque struct {
	// 计算参数
	edgeWidth int

	step int // 当c.EdgeWidth > 0, step = 2;

	initialTemperature float32 // 初始温度

	Field         *deque.ArrDeque
	thermalField  *deque.ArrDeque // 温度场容器
	thermalField1 *deque.ArrDeque

	// 每计算一个 ▲t 进行一次异或运算
	alternating bool

	//v int 拉速
	v        int64
	reminder int64

	calcHub *CalcHub

	// 状态
	isFlowing   bool // 是否有新的钢液注入
	isTail      bool // 拉尾坯
	isFull      bool // 铸机未充满
	isSeparated bool // 两种钢种

	e *executor

	mu sync.Mutex // 保护push data时对温度数据的并发访问
}

func NewCalculatorWithArrDeque(edgeWidth int) *calculatorWithArrDeque {
	c := &calculatorWithArrDeque{}
	if edgeWidth < 0 {
		edgeWidth = 0
	}
	if edgeWidth > 20 {
		edgeWidth = 20
	}

	start := time.Now()
	c.initialTemperature = 1550.0
	c.thermalField = deque.NewArrDeque(ZLength / ZStep)
	c.thermalField1 = deque.NewArrDeque(ZLength / ZStep)

	c.Field = c.thermalField

	c.v = int64(10 * 1.5 * 1000 / 60) // m / min
	c.alternating = true
	c.calcHub = NewCalcHub()

	c.edgeWidth = edgeWidth
	c.step = 1
	if c.edgeWidth > 0 {
		c.step = 2
	}

	initParameters()

	c.e = newExecutor(4, func(t task) {
		start := time.Now()
		count := 0
		c.Field.TraverseSpirally(t.start, t.end, func(z int, item *model.ItemType) {
			left, right, top, bottom := 0, Length/XStep-1, 0, Width/YStep-1 // 每个切片迭代时需要重置
			// 计算最外层， 逆时针
			// 1. 三个顶点，左下方顶点仅当其外一层温度不是初始温度时才开始计算
			//c.calculatePointLB(t.deltaT, z, item) 不用计算，因为还未产生温差
			{
				c.calculatePointRB(t.deltaT, z, item)
				c.calculatePointRT(t.deltaT, z, item)
				c.calculatePointLT(t.deltaT, z, item)
				count += 3
				for row := top + 1; row < bottom; row++ {
					// [row][right]
					c.calculatePointRA(t.deltaT, row, z, item)
					count++
				}
				for column := right - 1; column > left; column-- {
					// [bottom][column]
					c.calculatePointTA(t.deltaT, column, z, item)
					count++
				}
				right--
				bottom--
			}

			{
				//stop := 0 // 当出现两层的温度都是 初始温度时，则停止遍历
				//allNotChanged := true
				// 逆时针
				for left <= right && top <= bottom {
					c.calculatePointBA(t.deltaT, right, z, item)
					if item[0][right] != c.initialTemperature {
						//allNotChanged = false
					}
					count++
					for row := top + 1; row <= bottom; row++ {
						// [row][right]
						c.calculatePointIN(t.deltaT, right, row, z, item)
						if item[row][right] != c.initialTemperature {
							//allNotChanged = false
						}
						count++
					}
					if left < right && top < bottom {
						for column := right - 1; column > left; column-- {
							// [bottom][column]
							c.calculatePointIN(t.deltaT, column, bottom, z, item)
							if item[bottom][column] == c.initialTemperature {
								//allNotChanged = false
							}
							count++
						}
						c.calculatePointLA(t.deltaT, bottom, z, item)
						if item[bottom][0] != c.initialTemperature {
							//allNotChanged = false
						}
						count++
					}
					if top == bottom {
						c.calculatePointLB(t.deltaT, z, item)
						count++
						for column := right - 1; column > left; column-- {
							c.calculatePointBA(t.deltaT, column, z, item)
							count++
						}
					}
					right--
					bottom--
					// 如果该层与其的温度都未改变，即都是初始温度则计数
					//if allNotChanged {
					//	stop++
					//	allNotChanged = true
					//	if stop == 2 {
					//		break
					//	}
					//}
				}
			}
		})
		fmt.Println("消耗时间: ", time.Since(start), "计算的点数: ", count, "实际需要遍历的点数: ", (t.end - t.start) * 11340)
	})
	c.e.run()

	fmt.Println("初始化时间: ", time.Since(start))
	return c
}

func (c *calculatorWithArrDeque) GetCalcHub() *CalcHub {
	return c.calcHub
}

// 计算所有切片中最短的时间步长
func (c *calculatorWithArrDeque) calculateTimeStep() (float32, time.Duration) {
	start := time.Now()
	min := float32(1000.0)
	var t float32
	c.Field.Traverse(func(z int, item *model.ItemType) {
		t = calculateTimeStepOfOneSlice(item)
		if t < min {
			min = t
		}
	})

	//fmt.Println("计算deltaT花费的时间：", time.Since(start), min)
	return min, time.Since(start)
}

func (c *calculatorWithArrDeque) Run() {
	// 先计算timeStep
	duration := time.Second * 0
	count := 0
LOOP:
	for {
		if count > 100 {
			return
		}
		select {
		case <-c.calcHub.Stop:
			break LOOP
		default:
			deltaT, _ := c.calculateTimeStep()
			//calcDuration := c.calculateConcurrently(deltaT) // c.ThermalField.Field 最开始赋值为 ThermalField对应的指针
			calcDuration := c.calculateConcurrentlyBySlice(deltaT) // c.ThermalField.Field 最开始赋值为 ThermalField对应的指针
			if calcDuration < 25*time.Millisecond {
				calcDuration = 25 * time.Millisecond
			}
			duration += calcDuration
			// todo 这里需要根据准确的deltaT来确定时间步长
			if c.alternating {
				c.Field = c.thermalField1
			} else {
				c.Field = c.thermalField
			}

			c.updateSliceInfo(calcDuration)
			//if !c.Field.IsEmpty() {
			//	for i := Width/YStep - 1; i > Width/YStep-6; i-- {
			//		for j := Length/XStep - 5; j <= Length/XStep-1; j++ {
			//			fmt.Print(c.Field.Get(c.Field.Size() - 1, i, j), " ")
			//		}
			//		fmt.Print(i)
			//		fmt.Println()
			//	}
			//}
			c.alternating = !c.alternating // 仅在这里修改
			fmt.Println("计算温度场花费的时间：", duration)
			if duration > time.Second*4 {
				c.calcHub.PushSignal()
				count++
				duration = time.Second * 0
			}
		}
	}
}

func (c *calculatorWithArrDeque) updateSliceInfo(calcDuration time.Duration) {
	v := c.v // m/min -> mm/s
	var distance int64
	distance = v*calcDuration.Microseconds() + c.reminder
	if distance == 0 {
		return
	}
	c.reminder = distance % 1e7 // Microseconds = 1e6 and zStep = 10
	newSliceNum := distance / 1e7
	add := int(newSliceNum) // 加入的新切片数
	if c.isTail {
		// 处理拉尾坯的阶段
		fmt.Println("updateSliceInfo: 拉尾坯")
		for i := 0; i < add; i++ {
			c.thermalField.RemoveLast()
			c.thermalField1.RemoveLast()
		}
		// todo 处理不再进入新切片的情况，也需要考虑再次进入新切片时如何重新开始计算
		return
	}

	if c.isFull {
		fmt.Println("updateSliceInfo: 切片已满")
		// 新加入的切片未组成一个三维数组
		for i := 0; i < add; i++ {
			c.thermalField.RemoveLast()
			c.thermalField1.RemoveLast()
			c.thermalField.AddFirst(c.initialTemperature)
			c.thermalField1.AddFirst(c.initialTemperature)
		}
		for i := Width/YStep - 1; i > Width/YStep-6; i-- {
			for j := Length/XStep - 5; j <= Length/XStep-1; j++ {
				fmt.Print(c.Field.Get(c.Field.Size()-1, i, j), " ")
			}
			fmt.Print(i)
			fmt.Println()
		}
	} else {
		fmt.Println("updateSliceInfo: 切片未满")
		fmt.Println("updateSliceInfo: 新增切片数: ", add)
		for i := 0; i < add; i++ {
			if c.Field.IsFull() {
				c.thermalField.RemoveLast()
				c.thermalField1.RemoveLast()
				c.thermalField.AddFirst(c.initialTemperature)
				c.thermalField1.AddFirst(c.initialTemperature)
			} else {
				c.thermalField.AddFirst(c.initialTemperature)
				c.thermalField1.AddFirst(c.initialTemperature)
			}
		}
		if c.Field.IsFull() {
			c.isFull = true
		}
	}
	fmt.Println("updateSliceInfo 目前的切片数为：", c.Field.Size())
}

// 并行计算方法2
func (c *calculatorWithArrDeque) calculateConcurrentlyBySlice(deltaT float32) time.Duration {
	fmt.Println("calculate start")
	start := time.Now()
	c.e.start <- task{start: 0, end: c.Field.Size(), deltaT: deltaT}
	fmt.Println("task dispatched")
	<- c.e.finish
	fmt.Println("task finished")
	return time.Since(start)
}

// 并行计算方法1
func (c *calculatorWithArrDeque) calculateConcurrently(deltaT float32) time.Duration {
	var start = time.Now()
	var wg = sync.WaitGroup{}
	wg.Add(4)
	go func() {
		c.calculateCase1(deltaT)
		wg.Done()
	}()
	go func() {
		c.calculateCase2(deltaT)
		wg.Done()
	}()
	go func() {
		c.calculateCase3(deltaT)
		wg.Done()
	}()
	go func() {
		c.calculateCase4(deltaT)
		wg.Done()
	}()
	wg.Wait()
	//fmt.Println("并行计算时间：", time.Since(start))
	return time.Since(start)
}

func (c *calculatorWithArrDeque) BuildData() *TemperatureData {
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
		ThermalField.End = ZLength / ZStep
		ThermalField.IsFull = true
	} else {
		ThermalField.Start = 0
		ThermalField.End = z
	}
	fmt.Println("BuildData 温度场的长度：", z)
	if !c.Field.IsEmpty() {
		for i := Width/YStep - 1; i > Width/YStep-6; i-- {
			for j := Length/XStep - 5; j <= Length/XStep-1; j++ {
				fmt.Print(Field[z-1][i][j], " ")
			}
			fmt.Print(i, "build data")
			fmt.Println()
		}
	}

	buildDataHelper(ThermalField, temperatureData)
	temperatureData.Start = ThermalField.Start
	temperatureData.End = ThermalField.End
	temperatureData.IsFull = ThermalField.IsFull
	return temperatureData
}

// 并行计算
func (c *calculatorWithArrDeque) calculateCase1(deltaT float32) {
	//var start = time.Now()
	var count = 0
	c.Field.Traverse(func(z int, item *model.ItemType) {
		// 先计算点，再计算外表面，再计算里面的点
		c.calculatePointLT(deltaT, z, item)
		count++
		for i := 1; i < Length/XStep/2; i++ {
			c.calculatePointTA(deltaT, i, z, item)
			count++
		}
		for j := Width / YStep / 2; j < Width/YStep-1; j++ {
			c.calculatePointLA(deltaT, j, z, item)
			count++
		}
		for j := Width/YStep - 1 - c.edgeWidth; j < Width/YStep-1; j++ {
			for i := 1; i < 1+c.edgeWidth; i++ {
				c.calculatePointIN(deltaT, i, j, z, item)
				count++
			}
		}
		for j := Width / YStep / 2; j < Width/YStep-1-c.edgeWidth; j++ {
			for i := 1; i < 1+c.edgeWidth; i++ {
				c.calculatePointIN(deltaT, i, j, z, item)
				count++
			}
		}
		for j := Width/YStep - 1 - c.edgeWidth; j < Width/YStep-1; j++ {
			for i := 1 + c.edgeWidth; i < Length/XStep/2; i = i + 1 {
				c.calculatePointIN(deltaT, i, j, z, item)
				count++
			}
		}
		for j := Width / YStep / 2; j < Width/YStep-1-c.edgeWidth; j = j + c.step {
			for i := 1 + c.edgeWidth; i < Length/XStep/2; i = i + c.step {
				c.calculatePointIN(deltaT, i, j, z, item)
				count++
			}
		}
	})

	//fmt.Println("任务1执行时间: ", time.Since(start), "总共计算：", count, "个点")
}

func (c *calculatorWithArrDeque) calculateCase2(deltaT float32) {
	var start = time.Now()
	var count = 0
	c.Field.Traverse(func(z int, item *model.ItemType) {
		// 先计算点，再计算外表面，再计算里面的点
		c.calculatePointRT(deltaT, z, item)
		count++
		for i := Length / XStep / 2; i < Length/XStep-1; i++ {
			c.calculatePointTA(deltaT, i, z, item)
			count++
		}
		for j := Width / YStep / 2; j < Width/YStep-1; j++ {
			c.calculatePointRA(deltaT, j, z, item)
			count++
		}
		for j := Width/YStep - 1 - c.edgeWidth; j < Width/YStep-1; j++ {
			for i := Length/XStep - 1 - c.edgeWidth; i < Length/XStep-1; i++ {
				c.calculatePointIN(deltaT, i, j, z, item)
				count++
			}
		}
		for j := Width / YStep / 2; j < Width/YStep-1-c.edgeWidth; j++ {
			for i := Length/XStep - 1 - c.edgeWidth; i < Length/XStep-1; i++ {
				c.calculatePointIN(deltaT, i, j, z, item)
				count++
			}
		}
		for j := Width/YStep - 1 - c.edgeWidth; j < Width/YStep-1; j++ {
			for i := Length / XStep / 2; i < Length/XStep-1-c.edgeWidth; i = i + 1 {
				c.calculatePointIN(deltaT, i, j, z, item)
				count++
			}
		}
		for j := Width / YStep / 2; j < Width/YStep-1-c.edgeWidth; j = j + c.step {
			for i := Length / XStep / 2; i < Length/XStep-1-c.edgeWidth; i = i + c.step {
				c.calculatePointIN(deltaT, i, j, z, item)
				count++
			}
		}
	})
	fmt.Println("任务2执行时间: ", time.Since(start), "总共计算：", count, "个点")
}

func (c *calculatorWithArrDeque) calculateCase3(deltaT float32) {
	var start = time.Now()
	var count = 0
	c.Field.Traverse(func(z int, item *model.ItemType) {
		// 先计算点，再计算外表面，再计算里面的点
		c.calculatePointRB(deltaT, z, item)
		count++
		for i := Length / XStep / 2; i < Length/XStep-1; i++ {
			c.calculatePointBA(deltaT, i, z, item)
			count++
		}
		for j := 1; j < Width/YStep/2; j++ {
			c.calculatePointRA(deltaT, j, z, item)
			count++
		}
		for j := 1; j < 1+c.edgeWidth; j++ {
			for i := Length/XStep - 1 - c.edgeWidth; i < Length/XStep-1; i++ {
				c.calculatePointIN(deltaT, i, j, z, item)
				count++
			}
		}
		for j := 1 + c.edgeWidth; j < Width/YStep/2; j++ {
			for i := Length/XStep - 1 - c.edgeWidth; i < Length/XStep-1; i++ {
				c.calculatePointIN(deltaT, i, j, z, item)
				count++
			}
		}
		for j := 1; j < 1+c.edgeWidth; j++ {
			for i := Length / XStep / 2; i < Length/XStep-1-c.edgeWidth; i++ {
				c.calculatePointIN(deltaT, i, j, z, item)
				count++
			}
		}
		for j := 1 + c.edgeWidth; j < Width/YStep/2; j = j + c.step {
			for i := Length / XStep / 2; i < Length/XStep-1-c.edgeWidth; i = i + c.step {
				c.calculatePointIN(deltaT, i, j, z, item)
				count++
			}
		}
	})
	fmt.Println("任务3执行时间: ", time.Since(start), "总共计算：", count, "个点")
}

func (c *calculatorWithArrDeque) calculateCase4(deltaT float32) {
	var start = time.Now()
	var count = 0
	c.Field.Traverse(func(z int, item *model.ItemType) {
		// 先计算点，再计算外表面，再计算里面的点
		c.calculatePointLB(deltaT, z, item)
		count++
		for i := 1; i < Length/XStep/2; i++ {
			c.calculatePointBA(deltaT, i, z, item)
			count++
		}
		for j := 1; j < Width/YStep/2; j++ {
			c.calculatePointLA(deltaT, j, z, item)
			count++
		}
		for j := 1; j < 1+c.edgeWidth; j++ {
			for i := 1; i < 1+c.edgeWidth; i++ {
				c.calculatePointIN(deltaT, i, j, z, item)
				count++
			}
		}
		for j := 1 + c.edgeWidth; j < Width/YStep/2; j++ {
			for i := 1; i < 1+c.edgeWidth; i++ {
				c.calculatePointIN(deltaT, i, j, z, item)
				count++
			}
		}
		for j := 1; j < 1+c.edgeWidth; j++ {
			for i := 1 + c.edgeWidth; i < Length/XStep/2; i++ {
				c.calculatePointIN(deltaT, i, j, z, item)
				count++
			}
		}
		for j := 1 + c.edgeWidth; j < Width/YStep/2; j = j + c.step {
			for i := 1 + c.edgeWidth; i < Length/XStep/2; i = i + c.step {
				c.calculatePointIN(deltaT, i, j, z, item)
				count++
			}
		}
	})
	fmt.Println("任务4执行时间: ", time.Since(start), "总共计算：", count, "个点")
}

// 计算一个left top点的温度变化
func (c *calculatorWithArrDeque) calculatePointLT(deltaT float32, z int, slice *model.ItemType) {
	var index = int(slice[Width/YStep-1][0])/5 - 1
	var index1 = int(slice[Width/YStep-1][1])/5 - 1
	var index2 = int(slice[Width/YStep-2][0])/5 - 1
	var deltaHlt = getLambda(index, index1, 0, Width/YStep-1, 1, Width/YStep-1)*(slice[Width/YStep-1][0]-slice[Width/YStep-1][1])/float32(XStep*(getEx(1)+getEx(0))) +
		getLambda(index, index2, 0, Width/YStep-1, 0, Width/YStep-2)*(slice[Width/YStep-1][0]-slice[Width/YStep-2][0])/float32(YStep*(getEy(Width/YStep-2)+getEy(Width/YStep-1))) +
		Q[index]/(2*YStep)

	deltaHlt = deltaHlt * (2 * deltaT / Density[index])
	//fmt.Println(Thermalslice[Width/YStep-1][0]-Thermalslice[Width/YStep-1][1], Thermalslice[Width/YStep-1][0]-Thermalslice[Width/YStep-2][0], Q[index], deltaHlt/C[index], "左上角")

	if c.alternating {
		c.thermalField1.Set(z, Width/YStep-1, 0, slice[Width/YStep-1][0]-deltaHlt/C[index])
	} else {
		// 需要修改焓的变化到温度变化k映射关系
		c.thermalField.Set(z, Width/YStep-1, 0, slice[Width/YStep-1][0]-deltaHlt/C[index])
	}
}

// 计算上表面点温度变化
func (c *calculatorWithArrDeque) calculatePointTA(deltaT float32, x, z int, slice *model.ItemType) {
	var index = int(slice[Width/YStep-1][x])/5 - 1
	var index1 = int(slice[Width/YStep-1][x-1])/5 - 1
	var index2 = int(slice[Width/YStep-1][x+1])/5 - 1
	var index3 = int(slice[Width/YStep-2][x])/5 - 1
	var deltaHta = getLambda(index, index1, x, Width/YStep-1, x-1, Width/YStep-1)*(slice[Width/YStep-1][x]-slice[Width/YStep-1][x-1])/float32(XStep*(getEx(x-1)+getEx(x))) +
		getLambda(index, index2, x, Width/YStep-1, x+1, Width/YStep-1)*(slice[Width/YStep-1][x]-slice[Width/YStep-1][x+1])/float32(XStep*(getEx(x)+getEx(x+1))) +
		getLambda(index, index3, x, Width/YStep-1, x, Width/YStep-2)*(slice[Width/YStep-1][x]-slice[Width/YStep-2][x])/float32(YStep*(getEy(Width/YStep-2)+getEy(Width/YStep-1))) +
		Q[index]/(2*YStep)

	deltaHta = deltaHta * (2 * deltaT / Density[index])
	//fmt.Println(Thermalslice[Width/YStep-1][x]-Thermalslice[Width/YStep-1][x-1], Thermalslice[Width/YStep-1][x]-Thermalslice[Width/YStep-1][x+1], Thermalslice[Width/YStep-1][x]-Thermalslice[Width/YStep-2][x], Q[index], deltaHta/C[index], "上表面")

	if c.alternating {
		c.thermalField1.Set(z, Width/YStep-1, x, slice[Width/YStep-1][x]-deltaHta/C[index])
	} else {
		// 需要修改焓的变化到温度变化k映射关系
		c.thermalField.Set(z, Width/YStep-1, x, slice[Width/YStep-1][x]-deltaHta/C[index])
	}
}

// 计算right top点的温度变化
func (c *calculatorWithArrDeque) calculatePointRT(deltaT float32, z int, slice *model.ItemType) {
	var index = int(slice[Width/YStep-1][Length/XStep-1])/5 - 1
	var index1 = int(slice[Width/YStep-1][Length/XStep-2])/5 - 1
	var index2 = int(slice[Width/YStep-2][Length/XStep-1])/5 - 1
	var deltaHrt = getLambda(index, index1, Length/XStep-1, Width/YStep-1, Length/XStep-2, Width/YStep-1)*(slice[Width/YStep-1][Length/XStep-1]-slice[Width/YStep-1][Length/XStep-2])/float32(XStep*(getEx(Length/XStep-2)+getEx(Length/XStep-1))) +
		getLambda(index, index2, Length/XStep-1, Width/YStep-1, Length/XStep-1, Width/YStep-2)*(slice[Width/YStep-1][Length/XStep-1]-slice[Width/YStep-2][Length/XStep-1])/float32(YStep*(getEy(Width/YStep-2)+getEy(Width/YStep-1))) +
		Q[index]/(2*YStep) +
		Q[index]/(2*XStep)

	deltaHrt = deltaHrt * (2 * deltaT / Density[index])
	//fmt.Println(Thermalslice[Width/YStep-1][Length/XStep-1]-Thermalslice[Width/YStep-1][Length/XStep-2], Thermalslice[Width/YStep-1][Length/XStep-1]-Thermalslice[Width/YStep-2][Length/XStep-1], Q[index], deltaHrt/C[index],  "右上角")

	if c.alternating { // 需要修改焓的变化到温度变化的映射关系)
		c.thermalField1.Set(z, Width/YStep-1, Length/XStep-1, slice[Width/YStep-1][Length/XStep-1]-deltaHrt/C[index])
	} else {
		c.thermalField.Set(z, Width/YStep-1, Length/XStep-1, slice[Width/YStep-1][Length/XStep-1]-deltaHrt/C[index])
	}
}

// 计算右表面点的温度变化
func (c *calculatorWithArrDeque) calculatePointRA(deltaT float32, y, z int, slice *model.ItemType) {
	var index = int(slice[y][Length/XStep-1])/5 - 1
	var index1 = int(slice[y][Length/XStep-2])/5 - 1
	var index2 = int(slice[y-1][Length/XStep-1])/5 - 1
	var index3 = int(slice[y+1][Length/XStep-1])/5 - 1
	var deltaHra = getLambda(index, index1, Length/XStep-1, y, Length/XStep-2, y)*(slice[y][Length/XStep-1]-slice[y][Length/XStep-2])/float32(XStep*(getEx(Length/XStep-2)+getEx(Length/XStep-1))) +
		getLambda(index, index2, Length/XStep-1, y, Length/XStep-1, y-1)*(slice[y][Length/XStep-1]-slice[y-1][Length/XStep-1])/float32(YStep*(getEy(y-1)+getEy(y))) +
		getLambda(index, index3, Length/XStep-1, y, Length/XStep-1, y+1)*(slice[y][Length/XStep-1]-slice[y+1][Length/XStep-1])/float32(YStep*(getEy(y+1)+getEy(y))) +
		Q[index]/(2*XStep)

	deltaHra = deltaHra * (2 * deltaT / Density[index])
	//fmt.Println(Thermalslice[y][Length/XStep-1]-Thermalslice[y][Length/XStep-2], Thermalslice[y][Length/XStep-1]-Thermalslice[y-1][Length/XStep-1], Thermalslice[y][Length/XStep-1]-Thermalslice[y+1][Length/XStep-1], Q[index], deltaHra/C[index], "右表面")

	if c.alternating { // 需要修改焓的变化到温度变化的映射关系
		c.thermalField1.Set(z, y, Length/XStep-1, slice[y][Length/XStep-1]-deltaHra/C[index])
	} else {
		c.thermalField.Set(z, y, Length/XStep-1, slice[y][Length/XStep-1]-deltaHra/C[index])
	}
}

// 计算right bottom点的温度变化
func (c *calculatorWithArrDeque) calculatePointRB(deltaT float32, z int, slice *model.ItemType) {
	var index = int(slice[0][Length/XStep-1])/5 - 1
	var index1 = int(slice[0][Length/XStep-2])/5 - 1
	var index2 = int(slice[1][Length/XStep-1])/5 - 1
	var deltaHrb = getLambda(index, index1, Length/XStep-1, 0, Length/XStep-2, 0)*(slice[0][Length/XStep-1]-slice[0][Length/XStep-2])/float32(XStep*(getEx(Length/XStep-2)+getEx(Length/XStep-1))) +
		getLambda(index, index2, Length/XStep-1, 0, Length/XStep-1, 1)*(slice[0][Length/XStep-1]-slice[1][Length/XStep-1])/float32(YStep*(getEy(1)+getEy(0))) +
		Q[index]/(2*XStep)

	deltaHrb = deltaHrb * (2 * deltaT / Density[index])
	//fmt.Println(Thermalslice[0][Length/XStep-1]-Thermalslice[0][Length/XStep-2], Thermalslice[0][Length/XStep-1]-Thermalslice[1][Length/XStep-1], Q[index],deltaHrb/C[index], "右下角")

	if c.alternating { // 需要修改焓的变化到温度变化的映射关系
		c.thermalField1.Set(z, 0, Length/XStep-1, slice[0][Length/XStep-1]-deltaHrb/C[index])
	} else {
		c.thermalField.Set(z, 0, Length/XStep-1, slice[0][Length/XStep-1]-deltaHrb/C[index])
	}
}

// 计算下表面点的温度变化
func (c *calculatorWithArrDeque) calculatePointBA(deltaT float32, x, z int, slice *model.ItemType) {
	var index = int(slice[0][x])/5 - 1
	var index1 = int(slice[0][x-1])/5 - 1
	var index2 = int(slice[0][x+1])/5 - 1
	var index3 = int(slice[1][x])/5 - 1
	var deltaHba = getLambda(index, index1, x, 0, x-1, 0)*(slice[0][x]-slice[0][x-1])/float32(XStep*(getEx(x-1)+getEx(x))) +
		getLambda(index, index2, x, 0, x+1, 0)*(slice[0][x]-slice[0][x+1])/float32(XStep*(getEx(x+1)+getEx(x))) +
		getLambda(index, index3, x, 0, x, 1)*(slice[0][x]-slice[1][x])/float32(YStep*(getEy(1)+getEy(0)))

	deltaHba = deltaHba * (2 * deltaT / Density[index])
	//fmt.Println(Thermalslice[0][x]-Thermalslice[0][x-1], Thermalslice[0][x]-Thermalslice[0][x+1], Thermalslice[0][x]-Thermalslice[1][x],deltaHba/C[index], "下表面")

	if c.alternating { // 需要修改焓的变化到温度变化的映射关系)
		c.thermalField1.Set(z, 0, x, slice[0][x]-deltaHba/C[index])
	} else {
		c.thermalField.Set(z, 0, x, slice[0][x]-deltaHba/C[index])
	}
}

// 计算left bottom点的温度变化
func (c *calculatorWithArrDeque) calculatePointLB(deltaT float32, z int, slice *model.ItemType) {
	var index = int(slice[0][0])/5 - 1
	var index1 = int(slice[0][1])/5 - 1
	var index2 = int(slice[1][0])/5 - 1
	var deltaHlb = getLambda(index, index1, 1, 0, 0, 0)*(slice[0][0]-slice[0][1])/float32(XStep*(getEx(0)+getEx(1))) +
		getLambda(index, index2, 0, 1, 0, 0)*(slice[0][0]-slice[1][0])/float32(YStep*(getEy(1)+getEy(0)))

	deltaHlb = deltaHlb * (2 * deltaT / Density[index])
	//fmt.Println(Thermalslice[0][0]-Thermalslice[0][1], Thermalslice[0][0]-Thermalslice[1][0],deltaHlb/C[index], "左下角")

	if c.alternating { // 需要修改焓的变化到温度变化的映射关系)
		c.thermalField1.Set(z, 0, 0, slice[0][0]-deltaHlb/C[index])
	} else {
		c.thermalField.Set(z, 0, 0, slice[0][0]-deltaHlb/C[index])
	}
}

// 计算左表面点温度的变化
func (c *calculatorWithArrDeque) calculatePointLA(deltaT float32, y, z int, slice *model.ItemType) {
	var index = int(slice[y][0])/5 - 1
	var index1 = int(slice[y][1])/5 - 1
	var index2 = int(slice[y-1][0])/5 - 1
	var index3 = int(slice[y+1][0])/5 - 1
	var deltaHla = getLambda(index, index1, 1, y, 0, y)*(slice[y][0]-slice[y][1])/float32(XStep*(getEx(0)+getEx(1))) +
		getLambda(index, index2, 0, y-1, 0, y)*(slice[y][0]-slice[y-1][0])/float32(YStep*(getEy(y)+getEy(y-1))) +
		getLambda(index, index3, 0, y+1, 0, y)*(slice[y][0]-slice[y+1][0])/float32(YStep*(getEy(y)+getEy(y+1)))
	deltaHla = deltaHla * (2 * deltaT / Density[index])
	//fmt.Println(Thermalslice[y][0]-Thermalslice[y][1], Thermalslice[y][0]-Thermalslice[y-1][0], Thermalslice[y][0]-Thermalslice[y+1][0], deltaHla/C[index], "左表面")

	if c.alternating { // 需要修改焓的变化到温度变化的映射关系)
		c.thermalField1.Set(z, y, 0, slice[y][0]-deltaHla/C[index])
	} else {
		c.thermalField.Set(z, y, 0, slice[y][0]-deltaHla/C[index])
	}
}

// 计算内部点的温度变化
func (c *calculatorWithArrDeque) calculatePointIN(deltaT float32, x, y, z int, slice *model.ItemType) {
	var index = int(slice[y][x])/5 - 1
	var index1 = int(slice[y][x-1])/5 - 1
	var index2 = int(slice[y][x+1])/5 - 1
	var index3 = int(slice[y-1][x])/5 - 1
	var index4 = int(slice[y+1][x])/5 - 1
	var deltaHin = getLambda(index, index1, x-1, y, x, y)*(slice[y][x]-slice[y][x-1])/float32(XStep*(getEx(x)+getEx(x-1))) +
		getLambda(index, index2, x+1, y, x, y)*(slice[y][x]-slice[y][x+1])/float32(XStep*(getEx(x)+getEx(x+1))) +
		getLambda(index, index3, x, y-1, x, y)*(slice[y][x]-slice[y-1][x])/float32(YStep*(getEy(y)+getEy(y-1))) +
		getLambda(index, index4, x, y+1, x, y)*(slice[y][x]-slice[y+1][x])/float32(YStep*(getEy(y)+getEy(y+1)))
	deltaHin = deltaHin * (2 * deltaT / Density[index])
	//fmt.Println(Thermalslice[y][x]-Thermalslice[y][x-1], Thermalslice[y][x]-Thermalslice[y][x+1], Thermalslice[y][x]-Thermalslice[y-1][x], Thermalslice[y][x]-Thermalslice[y+1][x], deltaHin/C[index], deltaHin/C[index], "内部点")

	if c.alternating { // 需要修改焓的变化到温度变化的映射关系)
		c.thermalField1.Set(z, y, x, slice[y][x]-deltaHin/C[index])
	} else {
		c.thermalField.Set(z, y, x, slice[y][x]-deltaHin/C[index])
	}
}

func (c *calculatorWithArrDeque) Calculate() {
	for z := 0; z < 4000; z++ {
		c.thermalField.AddFirst(c.initialTemperature)
		c.thermalField1.AddFirst(c.initialTemperature)
	}

	start := time.Now()
	for count := 0; count < 100; count++ {
		deltaT, _ := c.calculateTimeStep()

		c.calculateConcurrently(deltaT)

		if c.alternating {
			c.Field = c.thermalField1
		} else {
			c.Field = c.thermalField
		}

		//for i := Width/YStep - 1; i > Width/YStep-6; i-- {
		//	for j := Length/XStep - 5; j <= Length/XStep-1; j++ {
		//		fmt.Print(math.Floor(float64(c.Field.Get(c.Field.Size()-1, i, j))), " ")
		//	}
		//	fmt.Print(i)
		//	fmt.Println()
		//}
		c.alternating = !c.alternating
	}

	fmt.Println("arr deque 总共消耗时间：", time.Since(start), "平均消耗时间: ", time.Since(start)/100)

	// 一个核心计算
	//c.CalculateSerially()
}
