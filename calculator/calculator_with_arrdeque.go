package calculator

import (
	"fmt"
	"lz/deque"
	"lz/model"
	"strconv"
	"sync"
	"time"
)

const (
	stateNotRunning          = 0
	stateRunning             = 1
	stateSuspended           = 2
	stateRunningWithTwoSteel = 3 // 存在两种钢种

	zone0 = 0 // 结晶区
	zone1 = 1 // 二冷区
)

var (
	oneSliceDuration time.Duration
)

type calculatorWithArrDeque struct {
	// 计算参数
	edgeWidth int

	step int // 当 c.EdgeWidth > 0, step = 2;

	initialTemperature float32 // 初始温度

	Field         *deque.ArrDeque
	thermalField  *deque.ArrDeque // 温度场容器
	thermalField1 *deque.ArrDeque

	// 每计算一个 ▲t 进行一次异或运算
	alternating bool

	// v int 拉速
	v        int64
	reminder int64

	calcHub *CalcHub

	// 状态
	runningState int  // 是否有铸坯还在铸机中
	isTail       bool // 拉尾坯
	isFull       bool // 铸机未充满

	start int
	end   int

	coolerConfig coolerConfig // 温度相关的配置
	r            int          // 结晶区与二冷区分界
	parameter1   *parameter   // 第一种钢种的物性参数
	parameter2   *parameter   // 第二种钢种的物性参数

	temperatureBottom float32 // 温度的下限

	e *executor

	mu sync.Mutex // 保护 push data时对温度数据的并发访问
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
	c.coolerConfig.StartTemperature = 1550.0 // 默认值
	c.thermalField = deque.NewArrDeque(model.ZLength / model.ZStep)
	c.thermalField1 = deque.NewArrDeque(model.ZLength / model.ZStep)

	c.Field = c.thermalField

	c.v = int64(10 * 1.5 * 1000 / 60) // m / min，默认速度1.5
	c.alternating = true
	c.calcHub = NewCalcHub()

	c.edgeWidth = edgeWidth
	c.step = 1
	if c.edgeWidth > 0 {
		c.step = 2
	}

	c.e = newExecutor(4, func(t task) {
		//start := time.Now()
		count := 0
		var parameter *parameter
		c.Field.TraverseSpirally(t.start, t.end, func(z int, item *model.ItemType) {
			// 跳过为空的切片， 即值为-1
			if item[0][0] == -1 {
				return
			}
			left, right, top, bottom := 0, model.Length/model.XStep-1, 0, model.Width/model.YStep-1 // 每个切片迭代时需要重置
			// parameter set
			parameter = c.getParameter(z)
			// 计算最外层， 逆时针
			// 1. 三个顶点，左下方顶点仅当其外一层温度不是初始温度时才开始计算
			//c.calculatePointLB(t.deltaT, z, item) 不用计算，因为还未产生温差
			{
				c.calculatePointRB(t.deltaT, z, item, parameter)
				c.calculatePointRT(t.deltaT, z, item, parameter)
				c.calculatePointLT(t.deltaT, z, item, parameter)
				count += 3
				for row := top + 1; row < bottom; row++ {
					// [row][right]
					c.calculatePointRA(t.deltaT, row, z, item, parameter)
					count++
				}
				for column := right - 1; column > left; column-- {
					// [bottom][column]
					c.calculatePointTA(t.deltaT, column, z, item, parameter)
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
					c.calculatePointBA(t.deltaT, right, z, item, parameter)
					if item[0][right] != c.coolerConfig.StartTemperature {
						//allNotChanged = false
					}
					count++
					for row := top + 1; row <= bottom; row++ {
						// [row][right]
						c.calculatePointIN(t.deltaT, right, row, z, item, parameter)
						if item[row][right] != c.coolerConfig.StartTemperature {
							//allNotChanged = false
						}
						count++
					}
					if left < right && top < bottom {
						for column := right - 1; column > left; column-- {
							// [bottom][column]
							c.calculatePointIN(t.deltaT, column, bottom, z, item, parameter)
							if item[bottom][column] == c.coolerConfig.StartTemperature {
								//allNotChanged = false
							}
							count++
						}
						c.calculatePointLA(t.deltaT, bottom, z, item, parameter)
						if item[bottom][0] != c.coolerConfig.StartTemperature {
							//allNotChanged = false
						}
						count++
					}
					if top == bottom {
						c.calculatePointLB(t.deltaT, z, item, parameter)
						count++
						for column := right - 1; column > left; column-- {
							c.calculatePointBA(t.deltaT, column, z, item, parameter)
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
		//fmt.Println("消耗时间: ", time.Since(start), "计算的点数: ", count, "实际需要遍历的点数: ", (t.end-t.start)*11340)
	})
	c.e.run() // 启动master线程分配任务，启动worker线程执行任务

	c.runningState = stateNotRunning // 未开始运行，只是完成初始化

	fmt.Println("初始化时间: ", time.Since(start))
	return c
}

func (c *calculatorWithArrDeque) InitParameter(steelValue int) {
	if c.runningState == stateRunning { // 如果此时有其他钢种正在计算，只有当拉尾坯模式将前一个铸坯全部移除铸机后，isRunning状态才会变为false
		// todo
	} else if c.runningState == stateSuspended {
		// todo
	} else {
		// 还未运行
		c.parameter1 = &parameter{} // 初始化
		initParameters(steelValue, c.parameter1)
	}
}

// 设置和冷却器有关的参数
func (c *calculatorWithArrDeque) SetCoolerConfig(env model.Env) {
	c.coolerConfig.StartTemperature = env.StartTemperature
	c.coolerConfig.NarrowSurfaceIn = env.NarrowSurfaceIn
	c.coolerConfig.NarrowSurfaceOut = env.NarrowSurfaceOut
	c.coolerConfig.WideSurfaceIn = env.WideSurfaceIn
	c.coolerConfig.WideSurfaceOut = env.WideSurfaceOut
	c.coolerConfig.SprayTemperature = env.SprayTemperature
	c.coolerConfig.RollerWaterTemperature = env.RollerWaterTemperature
	fmt.Println("设置冷却参数：", c.coolerConfig)
}

func (c *calculatorWithArrDeque) SetV(v float32) {
	c.v = int64(v * 1000 / 60)
	oneSliceDuration = time.Millisecond * time.Duration(1000*float32(model.ZStep)/float32(c.v)) // 10 / c.v
	fmt.Println("设置拉速：", c.v, v)
	fmt.Println("设置oneSliceDuration：", oneSliceDuration)
}

// 冷却器参数单独设置
func (c *calculatorWithArrDeque) SetStartTemperature(startTemperature float32) {
	c.coolerConfig.StartTemperature = startTemperature
}

func (c *calculatorWithArrDeque) SetNarrowSurfaceIn(narrowSurfaceIn float32) {
	c.coolerConfig.NarrowSurfaceIn = narrowSurfaceIn
}

func (c *calculatorWithArrDeque) SetNarrowSurfaceOut(narrowSurfaceOut float32) {
	c.coolerConfig.NarrowSurfaceOut = narrowSurfaceOut
}

func (c *calculatorWithArrDeque) SetWideSurfaceIn(wideSurfaceIn float32) {
	c.coolerConfig.WideSurfaceIn = wideSurfaceIn
}

func (c *calculatorWithArrDeque) SetWideSurfaceOut(wideSurfaceOut float32) {
	c.coolerConfig.WideSurfaceOut = wideSurfaceOut
}

func (c *calculatorWithArrDeque) SetSprayTemperature(sprayTemperature float32) {
	c.coolerConfig.SprayTemperature = sprayTemperature
}

func (c *calculatorWithArrDeque) SetRollerWaterTemperature(rollerWaterTemperature float32) {
	c.coolerConfig.RollerWaterTemperature = rollerWaterTemperature
}

func (c *calculatorWithArrDeque) GetCalcHub() *CalcHub {
	return c.calcHub
}

func (c *calculatorWithArrDeque) SetStateTail() {
	c.isTail = true
}

func (c *calculatorWithArrDeque) getFieldStart() int {
	return c.start
}

func (c *calculatorWithArrDeque) getFieldEnd() int {
	return c.end
}

func (c *calculatorWithArrDeque) whichZone(z int) int {
	z = z * model.ZStep / stepZ // stepZ代表Z方向的缩放比例
	if z <= upLength {          // upLength 代表结晶器的长度 R
		return zone0
	} else {
		// todo 不同的区域返回不同的代号
		return zone1
	}
}

func (c *calculatorWithArrDeque) getParameter(z int) *parameter {
	var parameter *parameter
	if c.runningState == stateRunning {
		parameter = c.parameter1
		if c.whichZone(z) == zone0 {
			parameter.GetHeff = c.getHeffLessThanR
			parameter.GetQ = c.getQLessThanR
			c.temperatureBottom = c.coolerConfig.NarrowSurfaceIn
		} else if c.whichZone(z) == zone1 {
			parameter.GetHeff = c.getHeffGreaterThanR
			parameter.GetQ = c.getQGreaterThanR
			c.temperatureBottom = c.coolerConfig.SprayTemperature
		}
	} else if c.runningState == stateRunningWithTwoSteel { // 处理两种钢种的情况
		// todo
		parameter = c.parameter2
	}
	return parameter
}

// 计算所有切片中最短的时间步长
func (c *calculatorWithArrDeque) calculateTimeStep() (float32, time.Duration) {
	start := time.Now()
	min := float32(1000.0)
	var t float32
	var parameter *parameter
	c.Field.Traverse(func(z int, item *model.ItemType) {
		// 跳过为空的切片
		if item[0][0] == -1 {
			return
		}
		// 根据 z 来确定 parameter c.getParameter(z)
		parameter = c.getParameter(z)
		t = calculateTimeStepOfOneSlice(item, parameter)
		if t < min {
			min = t
		}
	})

	//fmt.Println("计算deltaT花费的时间：", time.Since(start), min)
	return min, time.Since(start)
}

func (c *calculatorWithArrDeque) SliceDetailRun() {
LOOP:
	for {
		select {
		case <-c.calcHub.StopPushSliceDataSignalForRun:
			fmt.Println("stop slice detail running")
			c.calcHub.StopSuccessForRun <- struct{}{}
			break LOOP
		default:
			c.calcHub.PushSliceDetailSignal()
			time.Sleep(1 * time.Second)
		}
	}
}

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
	res.Marks[upLength] = "二冷区"
	return &res
}

func (c *calculatorWithArrDeque) GetFieldSize() int {
	return c.Field.Size()
}

func (c *calculatorWithArrDeque) Run() {
	c.runningState = stateRunning
	// 先计算timeStep
	duration := time.Second * 0
	//count := 0
LOOP:
	for {
		//if count > 100 {
		//	return
		//}
		select {
		case <-c.calcHub.Stop:
			c.runningState = stateSuspended
			break LOOP
		default:
			deltaT, _ := c.calculateTimeStep()
			//fmt.Println("deltaT: ", deltaT)
			//calcDuration := c.calculateConcurrently(deltaT) // c.ThermalField.Field 最开始赋值为 ThermalField对应的指针
			calcDuration := c.calculateConcurrentlyBySlice(deltaT) // c.ThermalField.Field 最开始赋值为 ThermalField对应的指针
			if calcDuration == 0 {                                 // 计算时间等于0，意味着还没有切片产生，此时可以等待产生一个切片再计算
				time.Sleep(oneSliceDuration)
				calcDuration = oneSliceDuration
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
			//	for i := model.Width/model.YStep - 1; i > model.Width/model.YStep-6; i-- {
			//		for j := model.Length/model.XStep - 5; j <= model.Length/model.XStep-1; j++ {
			//			fmt.Print(c.Field.Get(c.Field.Size()-1, i, j), " ")
			//		}
			//		fmt.Print(i)
			//		fmt.Println()
			//	}
			//}
			c.alternating = !c.alternating // 仅在这里修改
			//fmt.Println("计算温度场花费的时间：", duration)
			if duration > time.Second*4 {
				c.calcHub.PushSignal()
				//count++
				duration = time.Second * 0
			}
		}
	}
}

func (c *calculatorWithArrDeque) updateSliceInfo(calcDuration time.Duration) {
	v := c.v // m/min -> mm/s
	var distance int64
	distance = v*calcDuration.Microseconds() + c.reminder
	//fmt.Println("走过的距离: ", distance, c.v, calcDuration)
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
			if c.Field.IsFull() {
				c.thermalField.RemoveLast()
				c.thermalField.AddFirst(-1) // 使用-1代表该切片是空的
				c.thermalField1.RemoveLast()
				c.thermalField1.AddFirst(-1)
				// 当没有温度不为空的切片时，需要退出
				// 需要结合有没有新的钢种注入
				// 遍历时需要跳过为空的切片
			} else {
				c.thermalField.AddFirst(-1)
				c.thermalField1.AddFirst(-1)
			}
			if c.start < c.end {
				c.start++
			}
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
			c.thermalField.AddFirst(c.coolerConfig.StartTemperature)
			c.thermalField1.AddFirst(c.coolerConfig.StartTemperature)
		}
	} else {
		//fmt.Println("updateSliceInfo: 切片未满")
		//fmt.Println("updateSliceInfo: 新增切片数: ", add)
		for i := 0; i < add; i++ {
			if c.Field.IsFull() {
				c.thermalField.RemoveLast()
				c.thermalField1.RemoveLast()
				c.thermalField.AddFirst(c.coolerConfig.StartTemperature)
				c.thermalField1.AddFirst(c.coolerConfig.StartTemperature)
			} else {
				c.thermalField.AddFirst(c.coolerConfig.StartTemperature)
				c.thermalField1.AddFirst(c.coolerConfig.StartTemperature)
			}
			if c.end < model.ZLength/model.ZStep {
				c.end++
			}
		}
		if c.Field.IsFull() {
			c.isFull = true
		}
	}
	//fmt.Println("updateSliceInfo 目前的切片数为：", c.Field.Size())
}

// 并行计算方法2
func (c *calculatorWithArrDeque) calculateConcurrentlyBySlice(deltaT float32) time.Duration {
	//fmt.Println("calculate start")
	start := time.Now()
	c.e.start <- task{start: 0, end: c.Field.Size(), deltaT: deltaT}
	//fmt.Println("task dispatched")
	<-c.e.finish
	//fmt.Println("task finished")
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
	//	for i := model.Width/model.YStep - 1; i > model.Width/model.YStep-6; i-- {
	//		for j := model.Length/model.XStep - 5; j <= model.Length/model.XStep-1; j++ {
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

// 并行计算
func (c *calculatorWithArrDeque) calculateCase1(deltaT float32) {
	//var start = time.Now()
	var count = 0
	var parameter *parameter
	c.Field.Traverse(func(z int, item *model.ItemType) {
		// parameter set
		parameter = c.getParameter(z)
		// 先计算点，再计算外表面，再计算里面的点
		c.calculatePointLT(deltaT, z, item, parameter)
		count++
		for i := 1; i < model.Length/model.XStep/2; i++ {
			c.calculatePointTA(deltaT, i, z, item, parameter)
			count++
		}
		for j := model.Width / model.YStep / 2; j < model.Width/model.YStep-1; j++ {
			c.calculatePointLA(deltaT, j, z, item, parameter)
			count++
		}
		for j := model.Width/model.YStep - 1 - c.edgeWidth; j < model.Width/model.YStep-1; j++ {
			for i := 1; i < 1+c.edgeWidth; i++ {
				c.calculatePointIN(deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := model.Width / model.YStep / 2; j < model.Width/model.YStep-1-c.edgeWidth; j++ {
			for i := 1; i < 1+c.edgeWidth; i++ {
				c.calculatePointIN(deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := model.Width/model.YStep - 1 - c.edgeWidth; j < model.Width/model.YStep-1; j++ {
			for i := 1 + c.edgeWidth; i < model.Length/model.XStep/2; i = i + 1 {
				c.calculatePointIN(deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := model.Width / model.YStep / 2; j < model.Width/model.YStep-1-c.edgeWidth; j = j + c.step {
			for i := 1 + c.edgeWidth; i < model.Length/model.XStep/2; i = i + c.step {
				c.calculatePointIN(deltaT, i, j, z, item, parameter)
				count++
			}
		}
	})

	//fmt.Println("任务1执行时间: ", time.Since(start), "总共计算：", count, "个点")
}

func (c *calculatorWithArrDeque) calculateCase2(deltaT float32) {
	var start = time.Now()
	var count = 0
	var parameter *parameter
	c.Field.Traverse(func(z int, item *model.ItemType) {
		// parameter set
		parameter = c.getParameter(z)
		// 先计算点，再计算外表面，再计算里面的点
		c.calculatePointRT(deltaT, z, item, parameter)
		count++
		for i := model.Length / model.XStep / 2; i < model.Length/model.XStep-1; i++ {
			c.calculatePointTA(deltaT, i, z, item, parameter)
			count++
		}
		for j := model.Width / model.YStep / 2; j < model.Width/model.YStep-1; j++ {
			c.calculatePointRA(deltaT, j, z, item, parameter)
			count++
		}
		for j := model.Width/model.YStep - 1 - c.edgeWidth; j < model.Width/model.YStep-1; j++ {
			for i := model.Length/model.XStep - 1 - c.edgeWidth; i < model.Length/model.XStep-1; i++ {
				c.calculatePointIN(deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := model.Width / model.YStep / 2; j < model.Width/model.YStep-1-c.edgeWidth; j++ {
			for i := model.Length/model.XStep - 1 - c.edgeWidth; i < model.Length/model.XStep-1; i++ {
				c.calculatePointIN(deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := model.Width/model.YStep - 1 - c.edgeWidth; j < model.Width/model.YStep-1; j++ {
			for i := model.Length / model.XStep / 2; i < model.Length/model.XStep-1-c.edgeWidth; i = i + 1 {
				c.calculatePointIN(deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := model.Width / model.YStep / 2; j < model.Width/model.YStep-1-c.edgeWidth; j = j + c.step {
			for i := model.Length / model.XStep / 2; i < model.Length/model.XStep-1-c.edgeWidth; i = i + c.step {
				c.calculatePointIN(deltaT, i, j, z, item, parameter)
				count++
			}
		}
	})
	fmt.Println("任务2执行时间: ", time.Since(start), "总共计算：", count, "个点")
}

func (c *calculatorWithArrDeque) calculateCase3(deltaT float32) {
	var start = time.Now()
	var count = 0
	var parameter *parameter
	c.Field.Traverse(func(z int, item *model.ItemType) {
		// parameter set
		parameter = c.getParameter(z)
		// 先计算点，再计算外表面，再计算里面的点
		c.calculatePointRB(deltaT, z, item, parameter)
		count++
		for i := model.Length / model.XStep / 2; i < model.Length/model.XStep-1; i++ {
			c.calculatePointBA(deltaT, i, z, item, parameter)
			count++
		}
		for j := 1; j < model.Width/model.YStep/2; j++ {
			c.calculatePointRA(deltaT, j, z, item, parameter)
			count++
		}
		for j := 1; j < 1+c.edgeWidth; j++ {
			for i := model.Length/model.XStep - 1 - c.edgeWidth; i < model.Length/model.XStep-1; i++ {
				c.calculatePointIN(deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := 1 + c.edgeWidth; j < model.Width/model.YStep/2; j++ {
			for i := model.Length/model.XStep - 1 - c.edgeWidth; i < model.Length/model.XStep-1; i++ {
				c.calculatePointIN(deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := 1; j < 1+c.edgeWidth; j++ {
			for i := model.Length / model.XStep / 2; i < model.Length/model.XStep-1-c.edgeWidth; i++ {
				c.calculatePointIN(deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := 1 + c.edgeWidth; j < model.Width/model.YStep/2; j = j + c.step {
			for i := model.Length / model.XStep / 2; i < model.Length/model.XStep-1-c.edgeWidth; i = i + c.step {
				c.calculatePointIN(deltaT, i, j, z, item, parameter)
				count++
			}
		}
	})
	fmt.Println("任务3执行时间: ", time.Since(start), "总共计算：", count, "个点")
}

func (c *calculatorWithArrDeque) calculateCase4(deltaT float32) {
	var start = time.Now()
	var count = 0
	var parameter *parameter
	c.Field.Traverse(func(z int, item *model.ItemType) {
		// parameter set
		parameter = c.getParameter(z)
		// 先计算点，再计算外表面，再计算里面的点
		c.calculatePointLB(deltaT, z, item, parameter)
		count++
		for i := 1; i < model.Length/model.XStep/2; i++ {
			c.calculatePointBA(deltaT, i, z, item, parameter)
			count++
		}
		for j := 1; j < model.Width/model.YStep/2; j++ {
			c.calculatePointLA(deltaT, j, z, item, parameter)
			count++
		}
		for j := 1; j < 1+c.edgeWidth; j++ {
			for i := 1; i < 1+c.edgeWidth; i++ {
				c.calculatePointIN(deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := 1 + c.edgeWidth; j < model.Width/model.YStep/2; j++ {
			for i := 1; i < 1+c.edgeWidth; i++ {
				c.calculatePointIN(deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := 1; j < 1+c.edgeWidth; j++ {
			for i := 1 + c.edgeWidth; i < model.Length/model.XStep/2; i++ {
				c.calculatePointIN(deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := 1 + c.edgeWidth; j < model.Width/model.YStep/2; j = j + c.step {
			for i := 1 + c.edgeWidth; i < model.Length/model.XStep/2; i = i + c.step {
				c.calculatePointIN(deltaT, i, j, z, item, parameter)
				count++
			}
		}
	})
	fmt.Println("任务4执行时间: ", time.Since(start), "总共计算：", count, "个点")
}

// 计算一个left top点的温度变化
func (c *calculatorWithArrDeque) calculatePointLT(deltaT float32, z int, slice *model.ItemType, parameter *parameter) {
	var index = int(slice[model.Width/model.YStep-1][0]) - 1
	var index1 = int(slice[model.Width/model.YStep-1][1]) - 1
	var index2 = int(slice[model.Width/model.YStep-2][0]) - 1
	var deltaHlt = getLambda(index, index1, 0, model.Width/model.YStep-1, 1, model.Width/model.YStep-1, parameter)*(slice[model.Width/model.YStep-1][0]-slice[model.Width/model.YStep-1][1])/float32(model.XStep*(getEx(1)+getEx(0))) +
		getLambda(index, index2, 0, model.Width/model.YStep-1, 0, model.Width/model.YStep-2, parameter)*(slice[model.Width/model.YStep-1][0]-slice[model.Width/model.YStep-2][0])/float32(model.YStep*(getEy(model.Width/model.YStep-2)+getEy(model.Width/model.YStep-1))) +
		parameter.GetQ(slice[model.Width/model.YStep-1][0], parameter)/(2*model.YStep)

	deltaHlt = deltaHlt * (2 * deltaT / parameter.Density[index])
	//fmt.Println(Thermalslice[model.Width/model.YStep-1][0]-Thermalslice[model.Width/model.YStep-1][1], Thermalslice[model.Width/model.YStep-1][0]-Thermalslice[model.Width/model.YStep-2][0], Q[index], deltaHlt/C[index], "左上角")

	if c.alternating {
		c.thermalField1.Set(z, model.Width/model.YStep-1, 0, slice[model.Width/model.YStep-1][0]-deltaHlt/parameter.C[index], c.temperatureBottom)
	} else {
		// 需要修改焓的变化到温度变化k映射关系
		c.thermalField.Set(z, model.Width/model.YStep-1, 0, slice[model.Width/model.YStep-1][0]-deltaHlt/parameter.C[index], c.temperatureBottom)
	}
}

// 计算上表面点温度变化
func (c *calculatorWithArrDeque) calculatePointTA(deltaT float32, x, z int, slice *model.ItemType, parameter *parameter) {
	var index = int(slice[model.Width/model.YStep-1][x]) - 1
	var index1 = int(slice[model.Width/model.YStep-1][x-1]) - 1
	var index2 = int(slice[model.Width/model.YStep-1][x+1]) - 1
	var index3 = int(slice[model.Width/model.YStep-2][x]) - 1
	var deltaHta = getLambda(index, index1, x, model.Width/model.YStep-1, x-1, model.Width/model.YStep-1, parameter)*(slice[model.Width/model.YStep-1][x]-slice[model.Width/model.YStep-1][x-1])/float32(model.XStep*(getEx(x-1)+getEx(x))) +
		getLambda(index, index2, x, model.Width/model.YStep-1, x+1, model.Width/model.YStep-1, parameter)*(slice[model.Width/model.YStep-1][x]-slice[model.Width/model.YStep-1][x+1])/float32(model.XStep*(getEx(x)+getEx(x+1))) +
		getLambda(index, index3, x, model.Width/model.YStep-1, x, model.Width/model.YStep-2, parameter)*(slice[model.Width/model.YStep-1][x]-slice[model.Width/model.YStep-2][x])/float32(model.YStep*(getEy(model.Width/model.YStep-2)+getEy(model.Width/model.YStep-1))) +
		parameter.GetQ(slice[model.Width/model.YStep-1][x], parameter)/(2*model.YStep)

	deltaHta = deltaHta * (2 * deltaT / parameter.Density[index])
	//fmt.Println(deltaHta, "上表面")

	if c.alternating {
		c.thermalField1.Set(z, model.Width/model.YStep-1, x, slice[model.Width/model.YStep-1][x]-deltaHta/parameter.C[index], c.temperatureBottom)
	} else {
		// 需要修改焓的变化到温度变化k映射关系
		c.thermalField.Set(z, model.Width/model.YStep-1, x, slice[model.Width/model.YStep-1][x]-deltaHta/parameter.C[index], c.temperatureBottom)
	}
}

// 计算right top点的温度变化
func (c *calculatorWithArrDeque) calculatePointRT(deltaT float32, z int, slice *model.ItemType, parameter *parameter) {
	var index = int(slice[model.Width/model.YStep-1][model.Length/model.XStep-1]) - 1
	var index1 = int(slice[model.Width/model.YStep-1][model.Length/model.XStep-2]) - 1
	var index2 = int(slice[model.Width/model.YStep-2][model.Length/model.XStep-1]) - 1
	var deltaHrt = getLambda(index, index1, model.Length/model.XStep-1, model.Width/model.YStep-1, model.Length/model.XStep-2, model.Width/model.YStep-1, parameter)*(slice[model.Width/model.YStep-1][model.Length/model.XStep-1]-slice[model.Width/model.YStep-1][model.Length/model.XStep-2])/float32(model.XStep*(getEx(model.Length/model.XStep-2)+getEx(model.Length/model.XStep-1))) +
		getLambda(index, index2, model.Length/model.XStep-1, model.Width/model.YStep-1, model.Length/model.XStep-1, model.Width/model.YStep-2, parameter)*(slice[model.Width/model.YStep-1][model.Length/model.XStep-1]-slice[model.Width/model.YStep-2][model.Length/model.XStep-1])/float32(model.YStep*(getEy(model.Width/model.YStep-2)+getEy(model.Width/model.YStep-1))) +
		parameter.GetQ(slice[model.Width/model.YStep-1][model.Length/model.XStep-1], parameter)/(2*model.YStep) +
		parameter.GetQ(slice[model.Width/model.YStep-1][model.Length/model.XStep-1], parameter)/(2*model.XStep)

	deltaHrt = deltaHrt * (2 * deltaT / parameter.Density[index])
	//fmt.Println(Thermalslice[model.Width/model.YStep-1][model.Length/model.XStep-1]-Thermalslice[model.Width/model.YStep-1][model.Length/model.XStep-2], Thermalslice[model.Width/model.YStep-1][model.Length/model.XStep-1]-Thermalslice[model.Width/model.YStep-2][model.Length/model.XStep-1], Q[index], deltaHrt/C[index],  "右上角")
	if c.alternating { // 需要修改焓的变化到温度变化的映射关系)
		c.thermalField1.Set(z, model.Width/model.YStep-1, model.Length/model.XStep-1, slice[model.Width/model.YStep-1][model.Length/model.XStep-1]-deltaHrt/parameter.C[index], c.temperatureBottom)
	} else {
		c.thermalField.Set(z, model.Width/model.YStep-1, model.Length/model.XStep-1, slice[model.Width/model.YStep-1][model.Length/model.XStep-1]-deltaHrt/parameter.C[index], c.temperatureBottom)
	}
}

// 计算右表面点的温度变化
func (c *calculatorWithArrDeque) calculatePointRA(deltaT float32, y, z int, slice *model.ItemType, parameter *parameter) {
	var index = int(slice[y][model.Length/model.XStep-1]) - 1
	var index1 = int(slice[y][model.Length/model.XStep-2]) - 1
	var index2 = int(slice[y-1][model.Length/model.XStep-1]) - 1
	var index3 = int(slice[y+1][model.Length/model.XStep-1]) - 1
	var deltaHra = getLambda(index, index1, model.Length/model.XStep-1, y, model.Length/model.XStep-2, y, parameter)*(slice[y][model.Length/model.XStep-1]-slice[y][model.Length/model.XStep-2])/float32(model.XStep*(getEx(model.Length/model.XStep-2)+getEx(model.Length/model.XStep-1))) +
		getLambda(index, index2, model.Length/model.XStep-1, y, model.Length/model.XStep-1, y-1, parameter)*(slice[y][model.Length/model.XStep-1]-slice[y-1][model.Length/model.XStep-1])/float32(model.YStep*(getEy(y-1)+getEy(y))) +
		getLambda(index, index3, model.Length/model.XStep-1, y, model.Length/model.XStep-1, y+1, parameter)*(slice[y][model.Length/model.XStep-1]-slice[y+1][model.Length/model.XStep-1])/float32(model.YStep*(getEy(y+1)+getEy(y))) +
		parameter.GetQ(slice[y][model.Length/model.XStep-1], parameter)/(2*model.XStep)

	deltaHra = deltaHra * (2 * deltaT / parameter.Density[index])
	//fmt.Println(deltaHra, "右表面")
	//fmt.Println(getLambda(index, index1, model.Length/model.XStep-1, y, model.Length/model.XStep-2, y, parameter)*(slice[y][model.Length/model.XStep-1]-slice[y][model.Length/model.XStep-2])/float32(model.XStep*(getEx(model.Length/model.XStep-2)+getEx(model.Length/model.XStep-1))))
	//fmt.Println(getLambda(index, index2, model.Length/model.XStep-1, y, model.Length/model.XStep-1, y-1, parameter)*(slice[y][model.Length/model.XStep-1]-slice[y-1][model.Length/model.XStep-1])/float32(model.YStep*(getEy(y-1)+getEy(y))))
	//fmt.Println(getLambda(index, index3, model.Length/model.XStep-1, y, model.Length/model.XStep-1, y+1, parameter)*(slice[y][model.Length/model.XStep-1]-slice[y+1][model.Length/model.XStep-1])/float32(model.YStep*(getEy(y+1)+getEy(y))))
	//fmt.Println(parameter.GetQ(slice[y][model.Length/model.XStep-1], parameter)/(2*model.XStep))

	if c.alternating { // 需要修改焓的变化到温度变化的映射关系
		c.thermalField1.Set(z, y, model.Length/model.XStep-1, slice[y][model.Length/model.XStep-1]-deltaHra/parameter.C[index], c.temperatureBottom)
	} else {
		c.thermalField.Set(z, y, model.Length/model.XStep-1, slice[y][model.Length/model.XStep-1]-deltaHra/parameter.C[index], c.temperatureBottom)
	}
}

// 计算right bottom点的温度变化
func (c *calculatorWithArrDeque) calculatePointRB(deltaT float32, z int, slice *model.ItemType, parameter *parameter) {
	var index = int(slice[0][model.Length/model.XStep-1]) - 1
	var index1 = int(slice[0][model.Length/model.XStep-2]) - 1
	var index2 = int(slice[1][model.Length/model.XStep-1]) - 1
	var deltaHrb = getLambda(index, index1, model.Length/model.XStep-1, 0, model.Length/model.XStep-2, 0, parameter)*(slice[0][model.Length/model.XStep-1]-slice[0][model.Length/model.XStep-2])/float32(model.XStep*(getEx(model.Length/model.XStep-2)+getEx(model.Length/model.XStep-1))) +
		getLambda(index, index2, model.Length/model.XStep-1, 0, model.Length/model.XStep-1, 1, parameter)*(slice[0][model.Length/model.XStep-1]-slice[1][model.Length/model.XStep-1])/float32(model.YStep*(getEy(1)+getEy(0))) +
		parameter.GetQ(slice[0][model.Length/model.XStep-1], parameter)/(2*model.XStep)

	deltaHrb = deltaHrb * (2 * deltaT / parameter.Density[index])
	//fmt.Println(Thermalslice[0][model.Length/model.XStep-1]-Thermalslice[0][model.Length/model.XStep-2], Thermalslice[0][model.Length/model.XStep-1]-Thermalslice[1][model.Length/model.XStep-1], Q[index],deltaHrb/C[index], "右下角")

	if c.alternating { // 需要修改焓的变化到温度变化的映射关系
		c.thermalField1.Set(z, 0, model.Length/model.XStep-1, slice[0][model.Length/model.XStep-1]-deltaHrb/parameter.C[index], c.temperatureBottom)
	} else {
		c.thermalField.Set(z, 0, model.Length/model.XStep-1, slice[0][model.Length/model.XStep-1]-deltaHrb/parameter.C[index], c.temperatureBottom)
	}
}

// 计算下表面点的温度变化
func (c *calculatorWithArrDeque) calculatePointBA(deltaT float32, x, z int, slice *model.ItemType, parameter *parameter) {
	var index = int(slice[0][x]) - 1
	var index1 = int(slice[0][x-1]) - 1
	var index2 = int(slice[0][x+1]) - 1
	var index3 = int(slice[1][x]) - 1
	var deltaHba = getLambda(index, index1, x, 0, x-1, 0, parameter)*(slice[0][x]-slice[0][x-1])/float32(model.XStep*(getEx(x-1)+getEx(x))) +
		getLambda(index, index2, x, 0, x+1, 0, parameter)*(slice[0][x]-slice[0][x+1])/float32(model.XStep*(getEx(x+1)+getEx(x))) +
		getLambda(index, index3, x, 0, x, 1, parameter)*(slice[0][x]-slice[1][x])/float32(model.YStep*(getEy(1)+getEy(0)))

	deltaHba = deltaHba * (2 * deltaT / parameter.Density[index])
	//fmt.Println(Thermalslice[0][x]-Thermalslice[0][x-1], Thermalslice[0][x]-Thermalslice[0][x+1], Thermalslice[0][x]-Thermalslice[1][x],deltaHba/C[index], "下表面")

	if c.alternating { // 需要修改焓的变化到温度变化的映射关系)
		c.thermalField1.Set(z, 0, x, slice[0][x]-deltaHba/parameter.C[index], c.temperatureBottom)
	} else {
		c.thermalField.Set(z, 0, x, slice[0][x]-deltaHba/parameter.C[index], c.temperatureBottom)
	}
}

// 计算left bottom点的温度变化
func (c *calculatorWithArrDeque) calculatePointLB(deltaT float32, z int, slice *model.ItemType, parameter *parameter) {
	var index = int(slice[0][0]) - 1
	var index1 = int(slice[0][1]) - 1
	var index2 = int(slice[1][0]) - 1
	var deltaHlb = getLambda(index, index1, 1, 0, 0, 0, parameter)*(slice[0][0]-slice[0][1])/float32(model.XStep*(getEx(0)+getEx(1))) +
		getLambda(index, index2, 0, 1, 0, 0, parameter)*(slice[0][0]-slice[1][0])/float32(model.YStep*(getEy(1)+getEy(0)))

	deltaHlb = deltaHlb * (2 * deltaT / parameter.Density[index])
	//fmt.Println(Thermalslice[0][0]-Thermalslice[0][1], Thermalslice[0][0]-Thermalslice[1][0],deltaHlb/C[index], "左下角")

	if c.alternating { // 需要修改焓的变化到温度变化的映射关系)
		c.thermalField1.Set(z, 0, 0, slice[0][0]-deltaHlb/parameter.C[index], c.temperatureBottom)
	} else {
		c.thermalField.Set(z, 0, 0, slice[0][0]-deltaHlb/parameter.C[index], c.temperatureBottom)
	}
}

// 计算左表面点温度的变化
func (c *calculatorWithArrDeque) calculatePointLA(deltaT float32, y, z int, slice *model.ItemType, parameter *parameter) {
	var index = int(slice[y][0]) - 1
	var index1 = int(slice[y][1]) - 1
	var index2 = int(slice[y-1][0]) - 1
	var index3 = int(slice[y+1][0]) - 1
	var deltaHla = getLambda(index, index1, 1, y, 0, y, parameter)*(slice[y][0]-slice[y][1])/float32(model.XStep*(getEx(0)+getEx(1))) +
		getLambda(index, index2, 0, y-1, 0, y, parameter)*(slice[y][0]-slice[y-1][0])/float32(model.YStep*(getEy(y)+getEy(y-1))) +
		getLambda(index, index3, 0, y+1, 0, y, parameter)*(slice[y][0]-slice[y+1][0])/float32(model.YStep*(getEy(y)+getEy(y+1)))
	deltaHla = deltaHla * (2 * deltaT / parameter.Density[index])
	//fmt.Println(deltaHla, "左表面")

	if c.alternating { // 需要修改焓的变化到温度变化的映射关系)
		c.thermalField1.Set(z, y, 0, slice[y][0]-deltaHla/parameter.C[index], c.temperatureBottom)
	} else {
		c.thermalField.Set(z, y, 0, slice[y][0]-deltaHla/parameter.C[index], c.temperatureBottom)
	}
}

// 计算内部点的温度变化
func (c *calculatorWithArrDeque) calculatePointIN(deltaT float32, x, y, z int, slice *model.ItemType, parameter *parameter) {
	var index = int(slice[y][x]) - 1
	var index1 = int(slice[y][x-1]) - 1
	var index2 = int(slice[y][x+1]) - 1
	var index3 = int(slice[y-1][x]) - 1
	var index4 = int(slice[y+1][x]) - 1
	var deltaHin = getLambda(index, index1, x-1, y, x, y, parameter)*(slice[y][x]-slice[y][x-1])/float32(model.XStep*(getEx(x)+getEx(x-1))) +
		getLambda(index, index2, x+1, y, x, y, parameter)*(slice[y][x]-slice[y][x+1])/float32(model.XStep*(getEx(x)+getEx(x+1))) +
		getLambda(index, index3, x, y-1, x, y, parameter)*(slice[y][x]-slice[y-1][x])/float32(model.YStep*(getEy(y)+getEy(y-1))) +
		getLambda(index, index4, x, y+1, x, y, parameter)*(slice[y][x]-slice[y+1][x])/float32(model.YStep*(getEy(y)+getEy(y+1)))
	deltaHin = deltaHin * (2 * deltaT / parameter.Density[index])
	//fmt.Println(Thermalslice[y][x]-Thermalslice[y][x-1], Thermalslice[y][x]-Thermalslice[y][x+1], Thermalslice[y][x]-Thermalslice[y-1][x], Thermalslice[y][x]-Thermalslice[y+1][x], deltaHin/C[index], deltaHin/C[index], "内部点")
	//if x == model.Length / model.XStep - 4 && y == model.Width / model.YStep - 4 {
	//	fmt.Println(getLambda(index, index1, x-1, y, x, y, parameter)*(slice[y][x]-slice[y][x-1])/float32(model.XStep*(getEx(x)+getEx(x-1))), getLambda(index, index1, x-1, y, x, y, parameter), slice[y][x]-slice[y][x-1], float32(model.XStep*(getEx(x)+getEx(x-1))))
	//	fmt.Println(getLambda(index, index2, x+1, y, x, y, parameter)*(slice[y][x]-slice[y][x+1])/float32(model.XStep*(getEx(x)+getEx(x+1))), getLambda(index, index2, x+1, y, x, y, parameter), slice[y][x]-slice[y][x+1], float32(model.XStep*(getEx(x)+getEx(x+1))))
	//	fmt.Println(getLambda(index, index3, x, y-1, x, y, parameter)*(slice[y][x]-slice[y-1][x])/float32(model.YStep*(getEy(y)+getEy(y-1))), getLambda(index, index3, x, y-1, x, y, parameter), slice[y][x]-slice[y-1][x], float32(model.YStep*(getEy(y)+getEy(y-1))))
	//	fmt.Println(getLambda(index, index4, x, y+1, x, y, parameter)*(slice[y][x]-slice[y+1][x])/float32(model.YStep*(getEy(y)+getEy(y+1))), getLambda(index, index4, x, y+1, x, y, parameter), slice[y][x]-slice[y+1][x], float32(model.YStep*(getEy(y)+getEy(y+1))))
	//}
	if c.alternating { // 需要修改焓的变化到温度变化的映射关系)
		c.thermalField1.Set(z, y, x, slice[y][x]-deltaHin/parameter.C[index], c.temperatureBottom)
	} else {
		c.thermalField.Set(z, y, x, slice[y][x]-deltaHin/parameter.C[index], c.temperatureBottom)
	}
}

// 测试用
func (c *calculatorWithArrDeque) Calculate() {
	for z := 0; z < 4000; z++ {
		c.thermalField.AddFirst(c.coolerConfig.StartTemperature)
		c.thermalField1.AddFirst(c.coolerConfig.StartTemperature)
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
