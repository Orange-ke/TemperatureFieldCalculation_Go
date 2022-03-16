package calculator

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"lz/deque"
	"lz/model"
	"sync"
	"time"
)

const (
	stateNotRunning          = 0
	stateRunning             = 1
	stateSuspended           = 2
	stateRunningWithTwoSteel = 3 // 存在两种钢种
)

//var (
//	enthalpyScale = float32(100.0) / 4
//)

type calculatorWithArrDeque struct {
	// 计算参数
	Field         *deque.ArrDeque
	thermalField  *deque.ArrDeque // 温度场容器
	thermalField1 *deque.ArrDeque

	alternating bool // 每计算一个 ▲t 进行一次异或运算

	reminder int64 // 累计产生的切片的余数

	calcHub *CalcHub // 推送消息通道

	// 状态
	runningState int  // 是否有铸坯还在铸机中
	isTail       bool // 拉尾坯
	isFull       bool // 铸机未充满

	start int // 队列的开始位置
	end   int // 队列的结束位置

	castingMachine *CastingMachine // 铸机

	steel1 *Steel // 第一种钢种
	steel2 *Steel // 第二种钢种

	e executor

	mu sync.Mutex // 保护 push data时对温度数据的并发访问
}

func NewCalculatorWithArrDeque(e executor) *calculatorWithArrDeque {
	c := &calculatorWithArrDeque{}
	start := time.Now()
	// 初始化数据结构
	c.thermalField = deque.NewArrDeque(model.ZLength / model.ZStep)
	c.thermalField1 = deque.NewArrDeque(model.ZLength / model.ZStep)

	c.Field = c.thermalField
	c.alternating = true

	// 初始化铸机
	c.castingMachine = NewCastingMachine(1)

	// 初始化推送消息通道
	c.calcHub = NewCalcHub()
	if e == nil {
		c.e = newExecutorBaseOnSlice(8)
	} else {
		c.e = e
	}
	c.e.run(c) // 启动master线程分配任务，启动worker线程执行任务

	c.runningState = stateNotRunning // 未开始运行，只是完成初始化

	log.WithField("init_cost", time.Since(start)).Info("温度场计算器初始化耗时")
	return c
}

func (c *calculatorWithArrDeque) InitCastingMachine(castingMachineNumber int) {
	c.castingMachine = NewCastingMachine(castingMachineNumber)
}

func (c *calculatorWithArrDeque) GetCastingMachine() *CastingMachine {
	return c.castingMachine
}

func (c *calculatorWithArrDeque) InitSteel(steelValue int, castingMachine *CastingMachine) {
	if c.runningState == stateRunning { // 如果此时有其他钢种正在计算，只有当拉尾坯模式将前一个铸坯全部移除铸机后，isRunning状态才会变为false
		// todo
	} else if c.runningState == stateSuspended {
		// todo
	} else {
		// 还未运行
		c.steel1 = NewSteel(steelValue, castingMachine)
	}
}

func (c *calculatorWithArrDeque) getParameter(z int) *Parameter {
	if c.runningState == stateRunning {
		c.steel1.SetParameter(z)
		return c.steel1.Parameter
	} else if c.runningState == stateRunningWithTwoSteel { // 处理两种钢种的情况
		// todo
		// return
	}
	return nil
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

func (c *calculatorWithArrDeque) GetFieldSize() int {
	return c.Field.Size()
}

// 计算所有切片中最短的时间步长
func (c *calculatorWithArrDeque) calculateTimeStep() (float32, time.Duration) {
	start := time.Now()
	min := float32(1000.0)
	var t float32
	var parameter *Parameter
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

	fmt.Println("计算deltaT花费的时间：", time.Since(start), min)
	return min, time.Since(start)
}

func (c *calculatorWithArrDeque) Run() {
	c.runningState = stateRunning
	// 先计算timeStep
	var duration, calcDuration, gap time.Duration
	var deltaT float32
	//count := 0
LOOP:
	for {
		//if count > 10 {
		//	return
		//}
		select {
		case <-c.calcHub.Stop:
			c.runningState = stateSuspended
			break LOOP
		default:
			if c.Field.Size() == 0 { // 计算时间等于0，意味着还没有切片产生，此时可以等待产生一个切片再计算
				duration += OneSliceDuration
				gap = OneSliceDuration
				deltaT = float32(OneSliceDuration.Seconds())
			} else {
				deltaT, _ = c.calculateTimeStep()
				calcDuration = c.e.dispatchTask(deltaT, 0, c.Field.Size()) // c.ThermalField.Field 最开始赋值为 ThermalField对应的指针
				fmt.Println("计算单次时间：", calcDuration.Milliseconds())
				gap = time.Duration(int64(deltaT*1e9)) - calcDuration
				if gap < 0 {
					gap = 0
				}
				// todo
				duration += time.Duration(int64(deltaT * 1e9))
			}

			fmt.Println("时间步长: ", deltaT, gap, duration)
			time.Sleep(gap)
			// todo 这里需要根据准确的deltaT来确定时间步长
			if c.alternating {
				c.Field = c.thermalField1
			} else {
				c.Field = c.thermalField
			}

			c.updateSliceInfo(time.Duration(int64(deltaT * 1e9)))
			if !c.Field.IsEmpty() {
				for i := model.Width/model.YStep - 1; i > model.Width/model.YStep-6; i-- {
					for j := model.Length/model.XStep - 5; j <= model.Length/model.XStep-1; j++ {
						fmt.Print(c.Field.Get(c.Field.Size()-1, i, j), " ")
					}
					fmt.Print(i)
					fmt.Println()
				}
			}
			c.alternating = !c.alternating // 仅在这里修改
			log.WithFields(log.Fields{"deltaT": deltaT, "cost": duration}).Info("计算一次")
			if duration > time.Second*4 {
				//count++
				c.calcHub.PushSignal()
				duration = time.Second * 0
			}
		}
	}
}

func (c *calculatorWithArrDeque) updateSliceInfo(calcDuration time.Duration) {
	v := c.castingMachine.CoolerConfig.V // m/min -> mm/s
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
		log.Info("updateSliceInfo: 拉尾坯")
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
			c.thermalField.AddFirst(c.castingMachine.CoolerConfig.StartTemperature)
			c.thermalField1.AddFirst(c.castingMachine.CoolerConfig.StartTemperature)
		}
	} else {
		log.Info("切片未满, updateSliceInfo: 新增切片数:", add)
		for i := 0; i < add; i++ {
			if c.Field.IsFull() {
				c.thermalField.RemoveLast()
				c.thermalField1.RemoveLast()
				c.thermalField.AddFirst(c.castingMachine.CoolerConfig.StartTemperature)
				c.thermalField1.AddFirst(c.castingMachine.CoolerConfig.StartTemperature)
			} else {
				c.thermalField.AddFirst(c.castingMachine.CoolerConfig.StartTemperature)
				c.thermalField1.AddFirst(c.castingMachine.CoolerConfig.StartTemperature)
			}
			if c.end < model.ZLength/model.ZStep {
				c.end++
			}
		}
		if c.Field.IsFull() {
			c.isFull = true
		}
	}
	log.Info("updateSliceInfo 目前的切片数为：", c.Field.Size())
}

// 计算一个left top点的温度变化
func (c *calculatorWithArrDeque) calculatePointLT(deltaT float32, z int, slice *model.ItemType, parameter *Parameter) {
	var index = int(slice[model.Width/model.YStep-1][0]) - 1
	var index1 = int(slice[model.Width/model.YStep-1][1]) - 1
	var index2 = int(slice[model.Width/model.YStep-2][0]) - 1

	var deltaHlt = getLambda(index, index1, 0, model.Width/model.YStep-1, 1, model.Width/model.YStep-1, parameter)*(slice[model.Width/model.YStep-1][0]-slice[model.Width/model.YStep-1][1])/float32(stdXStep*(getEx(1)+getEx(0))) +
		getLambda(index, index2, 0, model.Width/model.YStep-1, 0, model.Width/model.YStep-2, parameter)*(slice[model.Width/model.YStep-1][0]-slice[model.Width/model.YStep-2][0])/float32(stdYStep*(getEy(model.Width/model.YStep-2)+getEy(model.Width/model.YStep-1))) +
		parameter.GetQ(slice[model.Width/model.YStep-1][0])/float32(2*stdYStep)
	deltaHlt = deltaHlt * (2 * deltaT / parameter.Density[index])
	//fmt.Println(
	//	getLambda(index, index1, 0, Width/YStep-1, 1, Width/YStep-1, parameter)*(slice[Width/YStep-1][0]-slice[Width/YStep-1][1])/(stdXStep*(getEx(1)+getEx(0))),
	//	getLambda(index, index2, 0, Width/YStep-1, 0, Width/YStep-2, parameter)*(slice[Width/YStep-1][0]-slice[Width/YStep-2][0])/(stdYStep*(getEy(Width/YStep-2)+getEy(Width/YStep-1))),
	//	parameter.GetQ(slice[Width/YStep-1][0])/(2*stdYStep),
	//	"left top",
	//)
	//fmt.Println("parameter.Enthalpy2Temp: ", parameter.Enthalpy2Temp, "deltaHrt:", deltaHlt, "△t:", deltaHlt*parameter.Enthalpy2Temp, "left top")

	if c.alternating {
		c.thermalField1.Set(z, model.Width/model.YStep-1, 0, slice[model.Width/model.YStep-1][0]-deltaHlt*parameter.Enthalpy2Temp, parameter.TemperatureBottom)
	} else {
		// 需要修改焓的变化到温度变化k映射关系
		c.thermalField.Set(z, model.Width/model.YStep-1, 0, slice[model.Width/model.YStep-1][0]-deltaHlt*parameter.Enthalpy2Temp, parameter.TemperatureBottom)
	}
}

// 计算上表面点温度变化
func (c *calculatorWithArrDeque) calculatePointTA(deltaT float32, x, z int, slice *model.ItemType, parameter *Parameter) {
	var index = int(slice[model.Width/model.YStep-1][x]) - 1
	var index1 = int(slice[model.Width/model.YStep-1][x-1]) - 1
	var index2 = int(slice[model.Width/model.YStep-1][x+1]) - 1
	var index3 = int(slice[model.Width/model.YStep-2][x]) - 1

	var deltaHta = getLambda(index, index1, x, model.Width/model.YStep-1, x-1, model.Width/model.YStep-1, parameter)*(slice[model.Width/model.YStep-1][x]-slice[model.Width/model.YStep-1][x-1])/float32(stdXStep*(getEx(x-1)+getEx(x))) +
		getLambda(index, index2, x, model.Width/model.YStep-1, x+1, model.Width/model.YStep-1, parameter)*(slice[model.Width/model.YStep-1][x]-slice[model.Width/model.YStep-1][x+1])/float32(stdXStep*(getEx(x)+getEx(x+1))) +
		getLambda(index, index3, x, model.Width/model.YStep-1, x, model.Width/model.YStep-2, parameter)*(slice[model.Width/model.YStep-1][x]-slice[model.Width/model.YStep-2][x])/float32(stdYStep*(getEy(model.Width/model.YStep-2)+getEy(model.Width/model.YStep-1))) +
		parameter.GetQ(slice[model.Width/model.YStep-1][x])/float32(2*stdYStep)
	deltaHta = deltaHta * (2 * deltaT / parameter.Density[index])
	//fmt.Println(
	//	getLambda(index, index1, x, Width/YStep-1, x-1, Width/YStep-1, parameter)*(slice[Width/YStep-1][x]-slice[Width/YStep-1][x-1])/(stdXStep*(getEx(x-1)+getEx(x))),
	//	getLambda(index, index2, x, Width/YStep-1, x+1, Width/YStep-1, parameter)*(slice[Width/YStep-1][x]-slice[Width/YStep-1][x+1])/(stdXStep*(getEx(x)+getEx(x+1))),
	//	getLambda(index, index3, x, Width/YStep-1, x, Width/YStep-2, parameter)*(slice[Width/YStep-1][x]-slice[Width/YStep-2][x])/(stdYStep*(getEy(Width/YStep-2)+getEy(Width/YStep-1))),
	//	parameter.GetQ(slice[Width/YStep-1][x])/(2*stdYStep),
	//	"top",
	//)
	//fmt.Println("parameter.Enthalpy2Temp: ", parameter.Enthalpy2Temp, "deltaHrt:", deltaHta, "△t:", deltaHta*parameter.Enthalpy2Temp, "top")

	if c.alternating {
		c.thermalField1.Set(z, model.Width/model.YStep-1, x, slice[model.Width/model.YStep-1][x]-deltaHta*parameter.Enthalpy2Temp, parameter.TemperatureBottom)
	} else {
		// 需要修改焓的变化到温度变化k映射关系
		c.thermalField.Set(z, model.Width/model.YStep-1, x, slice[model.Width/model.YStep-1][x]-deltaHta*parameter.Enthalpy2Temp, parameter.TemperatureBottom)
	}
}

// 计算right top点的温度变化
func (c *calculatorWithArrDeque) calculatePointRT(deltaT float32, z int, slice *model.ItemType, parameter *Parameter) {
	var index = int(slice[model.Width/model.YStep-1][model.Length/model.XStep-1]) - 1
	var index1 = int(slice[model.Width/model.YStep-1][model.Length/model.XStep-2]) - 1
	var index2 = int(slice[model.Width/model.YStep-2][model.Length/model.XStep-1]) - 1

	var deltaHrt = getLambda(index, index1, model.Length/model.XStep-1, model.Width/model.YStep-1, model.Length/model.XStep-2, model.Width/model.YStep-1, parameter)*(slice[model.Width/model.YStep-1][model.Length/model.XStep-1]-slice[model.Width/model.YStep-1][model.Length/model.XStep-2])/float32(stdXStep*(getEx(model.Length/model.XStep-2)+getEx(model.Length/model.XStep-1))) +
		getLambda(index, index2, model.Length/model.XStep-1, model.Width/model.YStep-1, model.Length/model.XStep-1, model.Width/model.YStep-2, parameter)*(slice[model.Width/model.YStep-1][model.Length/model.XStep-1]-slice[model.Width/model.YStep-2][model.Length/model.XStep-1])/float32(stdYStep*(getEy(model.Width/model.YStep-2)+getEy(model.Width/model.YStep-1))) +
		parameter.GetQ(slice[model.Width/model.YStep-1][model.Length/model.XStep-1])/float32(2*stdYStep) +
		parameter.GetQ(slice[model.Width/model.YStep-1][model.Length/model.XStep-1])/float32(2*stdXStep)
	deltaHrt = deltaHrt * (2 * deltaT / parameter.Density[index])
	//fmt.Println(
	//	getLambda(index, index1, Length/XStep-1, Width/YStep-1, Length/XStep-2, Width/YStep-1, parameter)*(slice[Width/YStep-1][Length/XStep-1]-slice[Width/YStep-1][Length/XStep-2])/(stdXStep*(getEx(Length/XStep-2)+getEx(Length/XStep-1))),
	//	getLambda(index, index2, Length/XStep-1, Width/YStep-1, Length/XStep-1, Width/YStep-2, parameter)*(slice[Width/YStep-1][Length/XStep-1]-slice[Width/YStep-2][Length/XStep-1])/(stdYStep*(getEy(Width/YStep-2)+getEy(Width/YStep-1))),
	//	parameter.GetQ(slice[Width/YStep-1][Length/XStep-1])/(2*stdYStep),
	//	parameter.GetQ(slice[Width/YStep-1][Length/XStep-1])/(2*stdXStep),
	//	"right top",
	//)
	//fmt.Println("parameter.Enthalpy2Temp: ", parameter.Enthalpy2Temp, "deltaHrt:", deltaHrt, "△t:", deltaHrt*parameter.Enthalpy2Temp, "right top")
	if c.alternating { // 需要修改焓的变化到温度变化的映射关系)
		c.thermalField1.Set(z, model.Width/model.YStep-1, model.Length/model.XStep-1, slice[model.Width/model.YStep-1][model.Length/model.XStep-1]-deltaHrt*parameter.Enthalpy2Temp, parameter.TemperatureBottom)
	} else {
		c.thermalField.Set(z, model.Width/model.YStep-1, model.Length/model.XStep-1, slice[model.Width/model.YStep-1][model.Length/model.XStep-1]-deltaHrt*parameter.Enthalpy2Temp, parameter.TemperatureBottom)
	}
}

// 计算右表面点的温度变化
func (c *calculatorWithArrDeque) calculatePointRA(deltaT float32, y, z int, slice *model.ItemType, parameter *Parameter) {
	var index = int(slice[y][model.Length/model.XStep-1]) - 1
	var index1 = int(slice[y][model.Length/model.XStep-2]) - 1
	var index2 = int(slice[y-1][model.Length/model.XStep-1]) - 1
	var index3 = int(slice[y+1][model.Length/model.XStep-1]) - 1

	var deltaHra = getLambda(index, index1, model.Length/model.XStep-1, y, model.Length/model.XStep-2, y, parameter)*(slice[y][model.Length/model.XStep-1]-slice[y][model.Length/model.XStep-2])/float32(stdXStep*(getEx(model.Length/model.XStep-2)+getEx(model.Length/model.XStep-1))) +
		getLambda(index, index2, model.Length/model.XStep-1, y, model.Length/model.XStep-1, y-1, parameter)*(slice[y][model.Length/model.XStep-1]-slice[y-1][model.Length/model.XStep-1])/float32(stdYStep*(getEy(y-1)+getEy(y))) +
		getLambda(index, index3, model.Length/model.XStep-1, y, model.Length/model.XStep-1, y+1, parameter)*(slice[y][model.Length/model.XStep-1]-slice[y+1][model.Length/model.XStep-1])/float32(stdYStep*(getEy(y+1)+getEy(y))) +
		parameter.GetQ(slice[y][model.Length/model.XStep-1])/float32(2*stdXStep)
	deltaHra = deltaHra * (2 * deltaT / parameter.Density[index])
	//fmt.Println(
	//	getLambda(index, index1, Length/XStep-1, y, Length/XStep-2, y, parameter)*(slice[y][Length/XStep-1]-slice[y][Length/XStep-2])/(stdXStep*(getEx(Length/XStep-2)+getEx(Length/XStep-1))),
	//	getLambda(index, index2, Length/XStep-1, y, Length/XStep-1, y-1, parameter)*(slice[y][Length/XStep-1]-slice[y-1][Length/XStep-1])/(stdYStep*(getEy(y-1)+getEy(y))),
	//	getLambda(index, index3, Length/XStep-1, y, Length/XStep-1, y+1, parameter)*(slice[y][Length/XStep-1]-slice[y+1][Length/XStep-1])/(stdYStep*(getEy(y+1)+getEy(y))),
	//	parameter.GetQ(slice[y][Length/XStep-1])/(2*stdXStep),
	//	"right",
	//)
	//fmt.Println("parameter.Enthalpy2Temp: ", parameter.Enthalpy2Temp, "deltaHrt:", deltaHra, "△t:", deltaHra*parameter.Enthalpy2Temp, "right")

	if c.alternating { // 需要修改焓的变化到温度变化的映射关系
		c.thermalField1.Set(z, y, model.Length/model.XStep-1, slice[y][model.Length/model.XStep-1]-deltaHra*parameter.Enthalpy2Temp, parameter.TemperatureBottom)
	} else {
		c.thermalField.Set(z, y, model.Length/model.XStep-1, slice[y][model.Length/model.XStep-1]-deltaHra*parameter.Enthalpy2Temp, parameter.TemperatureBottom)
	}
}

// 计算right bottom点的温度变化
func (c *calculatorWithArrDeque) calculatePointRB(deltaT float32, z int, slice *model.ItemType, parameter *Parameter) {
	var index = int(slice[0][model.Length/model.XStep-1]) - 1
	var index1 = int(slice[0][model.Length/model.XStep-2]) - 1
	var index2 = int(slice[1][model.Length/model.XStep-1]) - 1

	var deltaHrb = getLambda(index, index1, model.Length/model.XStep-1, 0, model.Length/model.XStep-2, 0, parameter)*(slice[0][model.Length/model.XStep-1]-slice[0][model.Length/model.XStep-2])/float32(stdXStep*(getEx(model.Length/model.XStep-2)+getEx(model.Length/model.XStep-1))) +
		getLambda(index, index2, model.Length/model.XStep-1, 0, model.Length/model.XStep-1, 1, parameter)*(slice[0][model.Length/model.XStep-1]-slice[1][model.Length/model.XStep-1])/float32(stdYStep*(getEy(1)+getEy(0))) +
		parameter.GetQ(slice[0][model.Length/model.XStep-1])/float32(2*stdXStep)
	deltaHrb = deltaHrb * (2 * deltaT / parameter.Density[index])

	//fmt.Println(
	//	getLambda(index, index1, Length/XStep-1, 0, Length/XStep-2, 0, parameter)*(slice[0][Length/XStep-1]-slice[0][Length/XStep-2])/(stdXStep*(getEx(Length/XStep-2)+getEx(Length/XStep-1))),
	//	getLambda(index, index2, Length/XStep-1, 0, Length/XStep-1, 1, parameter)*(slice[0][Length/XStep-1]-slice[1][Length/XStep-1])/(stdYStep*(getEy(1)+getEy(0))),
	//	parameter.GetQ(slice[0][Length/XStep-1])/(2*stdXStep),
	//	"right bottom",
	//)
	//fmt.Println("parameter.Enthalpy2Temp: ", parameter.Enthalpy2Temp, "deltaHrt:", deltaHrb, "△t:", deltaHrb*parameter.Enthalpy2Temp, "right bottom")
	if c.alternating { // 需要修改焓的变化到温度变化的映射关系
		c.thermalField1.Set(z, 0, model.Length/model.XStep-1, slice[0][model.Length/model.XStep-1]-deltaHrb*parameter.Enthalpy2Temp, parameter.TemperatureBottom)
	} else {
		c.thermalField.Set(z, 0, model.Length/model.XStep-1, slice[0][model.Length/model.XStep-1]-deltaHrb*parameter.Enthalpy2Temp, parameter.TemperatureBottom)
	}
}

// 计算下表面点的温度变化
func (c *calculatorWithArrDeque) calculatePointBA(deltaT float32, x, z int, slice *model.ItemType, parameter *Parameter) {
	var index = int(slice[0][x]) - 1
	var index1 = int(slice[0][x-1]) - 1
	var index2 = int(slice[0][x+1]) - 1
	var index3 = int(slice[1][x]) - 1

	var deltaHba = getLambda(index, index1, x, 0, x-1, 0, parameter)*(slice[0][x]-slice[0][x-1])/float32(stdXStep*(getEx(x-1)+getEx(x))) +
		getLambda(index, index2, x, 0, x+1, 0, parameter)*(slice[0][x]-slice[0][x+1])/float32(stdXStep*(getEx(x+1)+getEx(x))) +
		getLambda(index, index3, x, 0, x, 1, parameter)*(slice[0][x]-slice[1][x])/float32(stdYStep*(getEy(1)+getEy(0)))
	deltaHba = deltaHba * (2 * deltaT / parameter.Density[index])
	//fmt.Println(
	//	getLambda(index, index1, x, 0, x-1, 0, parameter)*(slice[0][x]-slice[0][x-1])/(stdXStep*(getEx(x-1)+getEx(x))),
	//	getLambda(index, index2, x, 0, x+1, 0, parameter)*(slice[0][x]-slice[0][x+1])/(stdXStep*(getEx(x+1)+getEx(x))),
	//	getLambda(index, index3, x, 0, x, 1, parameter)*(slice[0][x]-slice[1][x])/(stdYStep*(getEy(1)+getEy(0))),
	//	"bottom",
	//)
	//fmt.Println("parameter.Enthalpy2Temp: ", parameter.Enthalpy2Temp, "deltaHrt:", deltaHba, "△t:", deltaHba*parameter.Enthalpy2Temp, "bottom")
	if c.alternating { // 需要修改焓的变化到温度变化的映射关系)
		c.thermalField1.Set(z, 0, x, slice[0][x]-deltaHba*parameter.Enthalpy2Temp, parameter.TemperatureBottom)
	} else {
		c.thermalField.Set(z, 0, x, slice[0][x]-deltaHba*parameter.Enthalpy2Temp, parameter.TemperatureBottom)
	}
}

// 计算left bottom点的温度变化
func (c *calculatorWithArrDeque) calculatePointLB(deltaT float32, z int, slice *model.ItemType, parameter *Parameter) {
	var index = int(slice[0][0]) - 1
	var index1 = int(slice[0][1]) - 1
	var index2 = int(slice[1][0]) - 1

	var deltaHlb = getLambda(index, index1, 1, 0, 0, 0, parameter)*(slice[0][0]-slice[0][1])/float32(stdXStep*(getEx(0)+getEx(1))) +
		getLambda(index, index2, 0, 1, 0, 0, parameter)*(slice[0][0]-slice[1][0])/float32(stdYStep*(getEy(1)+getEy(0)))
	deltaHlb = deltaHlb * (2 * deltaT / parameter.Density[index])
	//fmt.Println(
	//	getLambda(index, index1, 1, 0, 0, 0, parameter)*(slice[0][0]-slice[0][1])/(stdXStep*(getEx(0)+getEx(1))),
	//	getLambda(index, index2, 0, 1, 0, 0, parameter)*(slice[0][0]-slice[1][0])/(stdYStep*(getEy(1)+getEy(0))),
	//	"left bottom",
	//)
	//fmt.Println("parameter.Enthalpy2Temp: ", parameter.Enthalpy2Temp, "deltaHrt:", deltaHlb, "△t:", deltaHlb*parameter.Enthalpy2Temp, "left bottom")
	if c.alternating { // 需要修改焓的变化到温度变化的映射关系)
		c.thermalField1.Set(z, 0, 0, slice[0][0]-deltaHlb*parameter.Enthalpy2Temp, parameter.TemperatureBottom)
	} else {
		c.thermalField.Set(z, 0, 0, slice[0][0]-deltaHlb*parameter.Enthalpy2Temp, parameter.TemperatureBottom)
	}
}

// 计算左表面点温度的变化
func (c *calculatorWithArrDeque) calculatePointLA(deltaT float32, y, z int, slice *model.ItemType, parameter *Parameter) {
	var index = int(slice[y][0]) - 1
	var index1 = int(slice[y][1]) - 1
	var index2 = int(slice[y-1][0]) - 1
	var index3 = int(slice[y+1][0]) - 1

	var deltaHla = getLambda(index, index1, 1, y, 0, y, parameter)*(slice[y][0]-slice[y][1])/float32(stdXStep*(getEx(0)+getEx(1))) +
		getLambda(index, index2, 0, y-1, 0, y, parameter)*(slice[y][0]-slice[y-1][0])/float32(stdYStep*(getEy(y)+getEy(y-1))) +
		getLambda(index, index3, 0, y+1, 0, y, parameter)*(slice[y][0]-slice[y+1][0])/float32(stdYStep*(getEy(y)+getEy(y+1)))
	deltaHla = deltaHla * (2 * deltaT / parameter.Density[index])
	//fmt.Println(
	//	getLambda(index, index1, 1, y, 0, y, parameter)*(slice[y][0]-slice[y][1])/(stdXStep*(getEx(0)+getEx(1))),
	//	getLambda(index, index2, 0, y-1, 0, y, parameter)*(slice[y][0]-slice[y-1][0])/(stdYStep*(getEy(y)+getEy(y-1))),
	//	getLambda(index, index3, 0, y+1, 0, y, parameter)*(slice[y][0]-slice[y+1][0])/(stdYStep*(getEy(y)+getEy(y+1))),
	//	"left",
	//)
	//fmt.Println("parameter.Enthalpy2Temp: ", parameter.Enthalpy2Temp, "deltaHrt:", deltaHla, "△t:", deltaHla*parameter.Enthalpy2Temp, "left")

	if c.alternating { // 需要修改焓的变化到温度变化的映射关系)
		c.thermalField1.Set(z, y, 0, slice[y][0]-deltaHla*parameter.Enthalpy2Temp, parameter.TemperatureBottom)
	} else {
		c.thermalField.Set(z, y, 0, slice[y][0]-deltaHla*parameter.Enthalpy2Temp, parameter.TemperatureBottom)
	}
}

// 计算内部点的温度变化
func (c *calculatorWithArrDeque) calculatePointIN(deltaT float32, x, y, z int, slice *model.ItemType, parameter *Parameter) {
	var index = int(slice[y][x]) - 1
	var index1 = int(slice[y][x-1]) - 1
	var index2 = int(slice[y][x+1]) - 1
	var index3 = int(slice[y-1][x]) - 1
	var index4 = int(slice[y+1][x]) - 1

	var deltaHin = getLambda(index, index1, x-1, y, x, y, parameter)*(slice[y][x]-slice[y][x-1])/float32(stdXStep*(getEx(x)+getEx(x-1))) +
		getLambda(index, index2, x+1, y, x, y, parameter)*(slice[y][x]-slice[y][x+1])/float32(stdXStep*(getEx(x)+getEx(x+1))) +
		getLambda(index, index3, x, y-1, x, y, parameter)*(slice[y][x]-slice[y-1][x])/float32(stdYStep*(getEy(y)+getEy(y-1))) +
		getLambda(index, index4, x, y+1, x, y, parameter)*(slice[y][x]-slice[y+1][x])/float32(stdYStep*(getEy(y)+getEy(y+1)))
	deltaHin = deltaHin * (2 * deltaT / parameter.Density[index])
	//if getLambda(index, index4, x, y+1, x, y, parameter)*(slice[y][x]-slice[y+1][x])/(stdYStep*(getEy(y)+getEy(y+1))) < 0 {
	//	fmt.Println(deltaHin, "in")
	//	fmt.Println(index, index1, index2, index3, index4)
	//	fmt.Println(
	//		getLambda(index, index1, x-1, y, x, y, parameter)*(slice[y][x]-slice[y][x-1])/(stdXStep*(getEx(x)+getEx(x-1))),
	//		getLambda(index, index2, x+1, y, x, y, parameter)*(slice[y][x]-slice[y][x+1])/(stdXStep*(getEx(x)+getEx(x+1))),
	//		getLambda(index, index3, x, y-1, x, y, parameter)*(slice[y][x]-slice[y-1][x])/(stdYStep*(getEy(y)+getEy(y-1))),
	//		getLambda(index, index4, x, y+1, x, y, parameter)*(slice[y][x]-slice[y+1][x])/(stdYStep*(getEy(y)+getEy(y+1))),
	//		"in",
	//	)
	//}
	//fmt.Println("parameter.Enthalpy2Temp: ", parameter.Enthalpy2Temp, "deltaHrt:", deltaHin, "△t:", deltaHin*parameter.Enthalpy2Temp, "in")
	if c.alternating { // 需要修改焓的变化到温度变化的映射关系)
		c.thermalField1.Set(z, y, x, slice[y][x]-deltaHin*parameter.Enthalpy2Temp, parameter.TemperatureBottom)
	} else {
		c.thermalField.Set(z, y, x, slice[y][x]-deltaHin*parameter.Enthalpy2Temp, parameter.TemperatureBottom)
	}
}

// 测试用
func NewCalculatorForGenerate() *calculatorWithArrDeque {
	c := &calculatorWithArrDeque{}
	// 初始化数据结构
	c.thermalField = deque.NewArrDeque(model.ZLength / model.ZStep)
	c.thermalField1 = deque.NewArrDeque(model.ZLength / model.ZStep)
	c.Field = c.thermalField
	return c
}

func (c *calculatorWithArrDeque) GenerateResult() *TemperatureFieldData {
	//zMax := ZLength / ZStep
	yMax := model.Width / model.YStep
	xMax := model.Length / model.XStep
	var minus float32
	initialTemp := float32(1600.0)
	var slice *model.ItemType
	var scale = float32(0.9665)
	var base1 = float32(664.864)
	for i := 0; i < 4000; i++ {
		c.Field.AddFirst(initialTemp)
	}
	for i := 4000 - 1; i >= 0; i-- {
		slice = c.Field.GetSlice(i)
		// 从右向左减少
		for y := yMax - 1; y >= 0; y-- {
			minus = base1 - base1*float32(4000-1-i)/float32(4000-1)
			for x := xMax - 1; x >= 0; x-- {
				minus *= scale
				slice[y][x] -= minus
			}
		}

		// 从上到下减少
		for x := xMax - 1; x >= 0; x-- {
			minus = base1 - base1*float32(4000-1-i)/float32(4000-1)
			for y := yMax - 1; y >= 0; y-- {
				minus *= scale
				slice[y][x] -= minus
			}
		}
	}

	var base2 = float32(1048.4)
	for i := 0; i < UpLength; i++ {
		slice = c.Field.GetSlice(i)
		// 从右向左减少
		for y := yMax - 1; y >= 0; y-- {
			minus = base2 - base2*float32(4000-1-i)/float32(4000-1)
			for x := xMax - 1; x >= 0; x-- {
				minus *= scale
				slice[y][x] -= minus
			}
		}

		// 从上到下减少
		for x := xMax - 1; x >= 0; x-- {
			minus = base2 - base2*float32(4000-1-i)/float32(4000-1)
			for y := yMax - 1; y >= 0; y-- {
				minus *= scale
				slice[y][x] -= minus
			}
		}
	}
	return c.BuildData()
}

type SliceInfo struct {
	HorizontalSolidThickness  int                                                                     `json:"horizontal_solid_thickness"`
	VerticalSolidThickness    int                                                                     `json:"vertical_solid_thickness"`
	HorizontalLiquidThickness int                                                                     `json:"horizontal_liquid_thickness"`
	VerticalLiquidThickness   int                                                                     `json:"vertical_liquid_thickness"`
	Slice                     *[model.Width / model.YStep * 2][model.Length / model.XStep * 2]float32 `json:"slice"`
}

func (c *calculatorWithArrDeque) GenerateSLiceInfo(index int) *SliceInfo {
	return c.buildSliceGenerateData(index)
}

func (c *calculatorWithArrDeque) buildSliceGenerateData(index int) *SliceInfo {
	solidTemp := float32(1445.69)
	liquidTemp := float32(1506.77)
	sliceInfo := &SliceInfo{}
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
	sliceInfo.Slice = &slice
	length := model.Length/model.XStep - 1
	width := model.Width/model.YStep - 1
	for i := length; i >= 0; i-- {
		if originData[0][i] <= solidTemp {
			sliceInfo.HorizontalSolidThickness = model.XStep * (length - i + 1)
		}
	}
	for i := length; i >= 0; i-- {
		if originData[0][i] >= liquidTemp {
			sliceInfo.HorizontalLiquidThickness = model.XStep * (length - i + 1)
			break
		}
	}

	for j := width; j >= 0; j-- {
		if originData[j][0] <= solidTemp {
			sliceInfo.VerticalSolidThickness = model.YStep * (width - j + 1)
		}
	}
	for j := width; j >= 0; j-- {
		if originData[j][0] >= liquidTemp {
			sliceInfo.VerticalLiquidThickness = model.YStep * (width - j + 1)
			break
		}
	}
	return sliceInfo
}

// 测试用
func (c *calculatorWithArrDeque) Calculate() {
	for z := 0; z < 4000; z++ {
		c.thermalField.AddFirst(c.castingMachine.CoolerConfig.StartTemperature)
		c.thermalField1.AddFirst(c.castingMachine.CoolerConfig.StartTemperature)
	}

	start := time.Now()
	for count := 0; count < 100; count++ {
		deltaT, _ := c.calculateTimeStep()

		cost := c.e.dispatchTask(deltaT, 0, c.Field.Size())

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
		fmt.Println("单此计算时间：", cost.Milliseconds())
	}

	fmt.Println("arr deque 总共消耗时间：", time.Since(start), "平均消耗时间: ", time.Since(start)/100)

	// 一个核心计算
	//c.CalculateSerially()
}