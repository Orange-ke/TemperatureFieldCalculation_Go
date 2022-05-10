package calculator

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"lz/deque"
	"lz/model"
	"math/rand"
	"sync"
	"time"
)

const (
	stateNotRunning          = 0
	stateRunning             = 1
	stateSuspended           = 2
	stateRunningWithTwoSteel = 3 // 存在两种钢种
)

var (
	// 实际铸坯尺寸
	Length  int
	XStep   = model.XStep
	Width   int
	YStep   = model.YStep
	ZLength int
	ZStep   = model.ZStep

	densityOfWater = float32(1000.0) // kg/m3
	cOfWater       = float32(4179.0)
)

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
	// 初始化铸机
	c.castingMachine = NewCastingMachine()

	// 初始化数据结构
	c.thermalField = deque.NewArrDeque(ZLength / ZStep)
	c.thermalField1 = deque.NewArrDeque(ZLength / ZStep)

	c.Field = c.thermalField
	c.alternating = true

	// 初始化推送消息通道
	c.calcHub = NewCalcHub()
	if e == nil {
		c.e = newExecutorBaseOnSlice(4)
	} else {
		c.e = e
	}
	c.e.run(c) // 启动master线程分配任务，启动worker线程执行任务

	c.runningState = stateNotRunning // 未开始运行，只是完成初始化

	log.WithField("init_cost", time.Since(start)).Info("温度场计算器初始化耗时")
	return c
}

func (c *calculatorWithArrDeque) InitCastingMachine() {
	c.castingMachine = NewCastingMachine()
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

func (c *calculatorWithArrDeque) InitPushData(coordinate model.Coordinate) {
	up := coordinate.CenterStartDistance + coordinate.LevelHeight - c.castingMachine.LevelHeight
	arc := coordinate.CenterEndDistance - coordinate.CenterStartDistance
	down := float32(coordinate.ZLength) - coordinate.CenterEndDistance
	initPushData(up, arc, down)
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
	min := bigNum
	var t float32
	var parameter *Parameter
	c.Field.Traverse(func(z int, item *model.ItemType) {
		// 跳过为空的切片
		if item[0][0] == -1 {
			return
		}
		// 根据 z 来确定 parameter c.getParameter(z)
		parameter = c.getParameter(z)
		t = calculateTimeStepOfOneSlice(z, item, parameter)
		if t < min {
			min = t
		}
	})
	if min >= 0.2 {
		min = 0.2
	}
	fmt.Println("计算deltaT花费的时间：", time.Since(start).Milliseconds(), min)
	return min, time.Since(start)
}

// 计算热流密度
func (c *calculatorWithArrDeque) calculateQ() {
	start := time.Now()
	if c.runningState == stateRunning {
		c.Field.Traverse(func(z int, item *model.ItemType) {
			initialQ := 1 / (ROfWater() + ROfCu() + 1/2220.0) * (item[Width/YStep-1][0] - c.castingMachine.CoolerConfig.WideSurfaceIn)
			j := 0
			for ; j < Length/XStep; j++ {
				if item[Width/YStep-1][j] > c.steel1.LiquidPhaseTemperature {
					c.steel1.Parameter.Q[z][j] = initialQ
				} else {
					break
				}
			}
			start := j - 1
			for ; j < Length/XStep; j++ {
				c.steel1.Parameter.Q[z][j] = initialQ - (initialQ*0.7)*float32(j-start)/float32(Length/XStep-1-start)
			}
			i := 0
			for ; i < Width/YStep; i++ {
				if item[i][Length/XStep-1] > c.steel1.LiquidPhaseTemperature {
					c.steel1.Parameter.Q[z][Length/XStep+Width/YStep-1-i] = initialQ
				} else {
					break
				}
			}
			start = i - 1
			for ; i < Width/YStep; i++ {
				c.steel1.Parameter.Q[z][Length/XStep+Width/YStep-1-i] = initialQ - (initialQ*0.7)*float32(i-start)/float32(Width/YStep-1-start)
			}
		})
	}
	fmt.Println("计算热流密度所需时间：", time.Since(start).Milliseconds())
}

func (c *calculatorWithArrDeque) calculateWideSurfaceEnergy(wideSurfaceH float32) float32 {
	var wideSurfaceEnergy float32
	var initialQ float32
	c.Field.Traverse(func(z int, item *model.ItemType) {
		initialQ = 1 / (ROfWater() + ROfCu() + 1/wideSurfaceH) * (item[Width/YStep-1][0] - c.castingMachine.CoolerConfig.WideSurfaceIn)
		j := 0
		for ; j < Length/XStep; j++ {
			if item[Width/YStep-1][j] > c.steel1.LiquidPhaseTemperature {
				c.steel1.Parameter.Q[z][j] = initialQ
				wideSurfaceEnergy += c.steel1.Parameter.Q[z][j] * float32(XStep*ZStep) / 1e6
			} else {
				break
			}
		}
		start := j - 1
		for ; j < Length/XStep; j++ {
			c.steel1.Parameter.Q[z][j] = initialQ - (initialQ*0.7)*float32(j-start)/float32(Length/XStep-1-start)
			wideSurfaceEnergy += c.steel1.Parameter.Q[z][j] * float32(XStep*ZStep) / 1e6
		}
	})
	return wideSurfaceEnergy
}

func (c *calculatorWithArrDeque) calculateNarrowSurfaceEnergy(narrowSurfaceH float32) float32 {
	var narrowSurfaceEnergy float32
	var initialQ float32
	c.Field.Traverse(func(z int, item *model.ItemType) {
		initialQ = 1 / (ROfWater() + ROfCu() + 1/narrowSurfaceH) * (item[0][Length/XStep-1] - c.castingMachine.CoolerConfig.NarrowSurfaceIn)
		i := 0
		for ; i < Width/YStep; i++ {
			if item[i][Length/XStep-1] > c.steel1.LiquidPhaseTemperature {
				c.steel1.Parameter.Q[z][Length/XStep+Width/YStep-1-i] = initialQ
				narrowSurfaceEnergy += c.steel1.Parameter.Q[z][Length/XStep+Width/YStep-1-i] * float32(YStep*ZStep) / 1e6
			} else {
				break
			}
		}
		start := i - 1
		for ; i < Width/YStep; i++ {
			c.steel1.Parameter.Q[z][Length/XStep+Width/YStep-1-i] = initialQ - (initialQ*0.7)*float32(i-start)/float32(Width/YStep-1-start)
			narrowSurfaceEnergy += c.steel1.Parameter.Q[z][Length/XStep+Width/YStep-1-i] * float32(YStep*ZStep) / 1e6
		}
	})
	return narrowSurfaceEnergy
}

// 在线计算Q
func (c *calculatorWithArrDeque) calculateQOnline() {
	start := time.Now()
	energyScale := float32(c.Field.Size()) / ((float32(c.castingMachine.MdLength) - c.castingMachine.LevelHeight) / float32(ZStep))
	fmt.Println("energyScale: ", energyScale)
	left, right := float32(500.0), float32(5000.0)
	var wideSurfaceH float32
	err := float32(0.0001)
	var calculateErr float32
	targetWideSurfaceEnergy := c.castingMachine.CoolerConfig.WideWaterVolume / 1000 / 60 / 2 * densityOfWater * cOfWater * (c.castingMachine.CoolerConfig.WideSurfaceOut - c.castingMachine.CoolerConfig.WideSurfaceIn) * energyScale
	for left < right {
		wideSurfaceH = left + (right-left)/2
		calculateErr = 1 - c.calculateWideSurfaceEnergy(wideSurfaceH)/targetWideSurfaceEnergy
		if abs(calculateErr) <= err {
			break
		} else {
			if calculateErr < 0 {
				right = wideSurfaceH - 1
			} else {
				left = wideSurfaceH + 1
			}
		}
	}
	fmt.Println("targetWideSurfaceEnergy:", targetWideSurfaceEnergy, "wideSurfaceEnergy:", c.calculateWideSurfaceEnergy(wideSurfaceH), wideSurfaceH)

	var narrowSurfaceH float32
	targetNarrowSurfaceEnergy := c.castingMachine.CoolerConfig.NarrowWaterVolume / 1000 / 60 / 2 * densityOfWater * cOfWater * (c.castingMachine.CoolerConfig.NarrowSurfaceOut - c.castingMachine.CoolerConfig.NarrowSurfaceIn) * energyScale
	left, right = float32(500.0), float32(5000.0)
	for left < right {
		narrowSurfaceH = left + (right-left)/2
		calculateErr = 1 - c.calculateNarrowSurfaceEnergy(narrowSurfaceH)/targetNarrowSurfaceEnergy
		if abs(calculateErr) <= err {
			break
		} else {
			if calculateErr < 0 {
				right = narrowSurfaceH - 1
			} else {
				left = narrowSurfaceH + 1
			}
		}
	}
	fmt.Println("targetNarrowSurfaceEnergy:", targetNarrowSurfaceEnergy, "narrowSurfaceEnergy:", c.calculateNarrowSurfaceEnergy(narrowSurfaceH), narrowSurfaceH)

	fmt.Println("计算热流密度所需时间：", time.Since(start).Milliseconds())
}

// 计算综合换热系数
func (c *calculatorWithArrDeque) calculateHeff() {
	start := time.Now()
	if c.runningState == stateRunning {
		c.Field.Traverse(func(z int, item *model.ItemType) {
			for j := 0; j < Length/XStep; j++ {
				c.steel1.Parameter.Heff[z][j] = c.steel1.Parameter.Q[z][j] / (item[Width/YStep-1][j] - c.castingMachine.CoolerConfig.WideSurfaceIn)
			}
			for i := 0; i < Width/YStep; i++ {
				c.steel1.Parameter.Heff[z][Length/XStep+i] = c.steel1.Parameter.Q[z][Length/XStep+i] / (item[i][Length/XStep-1] - c.castingMachine.CoolerConfig.WideSurfaceIn)
			}
		})
	}
	fmt.Println("计算综合换热系数所需时间：", time.Since(start).Milliseconds())
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
		// 目前只计算到结晶器结束
		if c.Field.Size() >= 85 {
			return
		}
		select {
		case <-c.calcHub.Stop:
			c.runningState = stateSuspended
			break LOOP
		default:
			if c.Field.Size() == 0 { // 计算时间等于0，意味着还没有切片产生，此时可以等待产生一个切片再计算
				log.Info("切片数为0，此时直接生成一个切片")
				duration += OneSliceDuration
				gap = OneSliceDuration
				deltaT = float32(OneSliceDuration.Seconds())
			} else {
				c.calculateQOnline()
				c.calculateHeff()
				fmt.Println("Q: ", c.steel1.Parameter.Q[c.Field.Size()-1][:Length/XStep+Width/YStep])
				fmt.Println("Heff: ", c.steel1.Parameter.Heff[c.Field.Size()-1][:Length/XStep+Width/YStep])
				deltaT, _ = c.calculateTimeStep()
				calcDuration = c.e.dispatchTask(deltaT, 0, c.Field.Size()) // c.ThermalField.Field 最开始赋值为 ThermalField对应的指针
				fmt.Println("计算单次时间：", calcDuration.Milliseconds(), "ms")
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
				for i := Width/YStep - 1; i >= 0; i-- {
					for j := 0; j <= Length/XStep-1; j++ {
						fmt.Printf("%.2f ", c.Field.Get(c.Field.Size()-1, i, j))
					}
					fmt.Println()
				}
			}
			c.alternating = !c.alternating // 仅在这里修改
			log.WithFields(log.Fields{"deltaT": deltaT, "cost": duration.Milliseconds()}).Info("计算一次")
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
		// todo 处理不再进入新切片的情况
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
			if c.end < ZLength/ZStep {
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
	var index = int(slice[Width/YStep-1][0]) - 1
	var index1 = int(slice[Width/YStep-1][1]) - 1
	var index2 = int(slice[Width/YStep-2][0]) - 1
	// 求焓变
	var deltaHlt = getLambda(index, index1, 0, Width/YStep-1, 1, Width/YStep-1, parameter)*(slice[Width/YStep-1][0]-slice[Width/YStep-1][1])/(stdXStep*(getEx(1)+getEx(0))) +
		getLambda(index, index2, 0, Width/YStep-1, 0, Width/YStep-2, parameter)*(slice[Width/YStep-1][0]-slice[Width/YStep-2][0])/(stdYStep*(getEy(Width/YStep-2)+getEy(Width/YStep-1))) +
		parameter.GetQ(0, Width/YStep-1, z)/(2*stdYStep)
	deltaHlt = deltaHlt * (2 * deltaT / parameter.Density[index])
	//fmt.Println(
	//	getLambda(index, index1, 0, Width/YStep-1, 1, Width/YStep-1, parameter)*(slice[Width/YStep-1][0]-slice[Width/YStep-1][1])/(stdXStep*(getEx(1)+getEx(0))),
	//	getLambda(index, index2, 0, Width/YStep-1, 0, Width/YStep-2, parameter)*(slice[Width/YStep-1][0]-slice[Width/YStep-2][0])/(stdYStep*(getEy(Width/YStep-2)+getEy(Width/YStep-1))),
	//	parameter.GetQ(slice[Width/YStep-1][0])/(2*stdYStep),
	//	"left top",
	//)
	//fmt.Println("parameter.Enthalpy2Temp: ", parameter.Enthalpy2Temp, "deltaHrt:", deltaHlt, "△t:", deltaHlt*parameter.Enthalpy2Temp, "left top")
	targetTemp := parameter.Enthalpy2Temp(parameter.Temp2Enthalpy(slice[Width/YStep-1][0]) - deltaHlt)
	if c.alternating {
		c.thermalField1.Set(z, Width/YStep-1, 0, targetTemp, parameter.TemperatureBottom)
	} else {
		// 需要修改焓的变化到温度变化k映射关系
		c.thermalField.Set(z, Width/YStep-1, 0, targetTemp, parameter.TemperatureBottom)
	}
}

// 计算上表面点温度变化
func (c *calculatorWithArrDeque) calculatePointTA(deltaT float32, x, z int, slice *model.ItemType, parameter *Parameter) {
	var index = int(slice[Width/YStep-1][x]) - 1
	var index1 = int(slice[Width/YStep-1][x-1]) - 1
	var index2 = int(slice[Width/YStep-1][x+1]) - 1
	var index3 = int(slice[Width/YStep-2][x]) - 1

	var deltaHta = getLambda(index, index1, x, Width/YStep-1, x-1, Width/YStep-1, parameter)*(slice[Width/YStep-1][x]-slice[Width/YStep-1][x-1])/(stdXStep*(getEx(x-1)+getEx(x))) +
		getLambda(index, index2, x, Width/YStep-1, x+1, Width/YStep-1, parameter)*(slice[Width/YStep-1][x]-slice[Width/YStep-1][x+1])/(stdXStep*(getEx(x)+getEx(x+1))) +
		getLambda(index, index3, x, Width/YStep-1, x, Width/YStep-2, parameter)*(slice[Width/YStep-1][x]-slice[Width/YStep-2][x])/(stdYStep*(getEy(Width/YStep-2)+getEy(Width/YStep-1))) +
		parameter.GetQ(x, Width/YStep-1, z)/(2*stdYStep)
	deltaHta = deltaHta * (2 * deltaT / parameter.Density[index])
	//fmt.Println(
	//	getLambda(index, index1, x, Width/YStep-1, x-1, Width/YStep-1, parameter)*(slice[Width/YStep-1][x]-slice[Width/YStep-1][x-1])/(stdXStep*(getEx(x-1)+getEx(x))),
	//	getLambda(index, index2, x, Width/YStep-1, x+1, Width/YStep-1, parameter)*(slice[Width/YStep-1][x]-slice[Width/YStep-1][x+1])/(stdXStep*(getEx(x)+getEx(x+1))),
	//	getLambda(index, index3, x, Width/YStep-1, x, Width/YStep-2, parameter)*(slice[Width/YStep-1][x]-slice[Width/YStep-2][x])/(stdYStep*(getEy(Width/YStep-2)+getEy(Width/YStep-1))),
	//	parameter.GetQ(slice[Width/YStep-1][x])/(2*stdYStep),
	//	"top",
	//)
	//fmt.Println("parameter.Enthalpy2Temp: ", parameter.Enthalpy2Temp, "deltaHrt:", deltaHta, "△t:", deltaHta*parameter.Enthalpy2Temp, "top")
	targetTemp := parameter.Enthalpy2Temp(parameter.Temp2Enthalpy(slice[Width/YStep-1][x]) - deltaHta)
	if c.alternating {
		c.thermalField1.Set(z, Width/YStep-1, x, targetTemp, parameter.TemperatureBottom)
	} else {
		// 需要修改焓的变化到温度变化k映射关系
		c.thermalField.Set(z, Width/YStep-1, x, targetTemp, parameter.TemperatureBottom)
	}
}

// 计算right top点的温度变化
func (c *calculatorWithArrDeque) calculatePointRT(deltaT float32, z int, slice *model.ItemType, parameter *Parameter) {
	var index = int(slice[Width/YStep-1][Length/XStep-1]) - 1
	var index1 = int(slice[Width/YStep-1][Length/XStep-2]) - 1
	var index2 = int(slice[Width/YStep-2][Length/XStep-1]) - 1

	var deltaHrt = getLambda(index, index1, Length/XStep-1, Width/YStep-1, Length/XStep-2, Width/YStep-1, parameter)*(slice[Width/YStep-1][Length/XStep-1]-slice[Width/YStep-1][Length/XStep-2])/(stdXStep*(getEx(Length/XStep-2)+getEx(Length/XStep-1))) +
		getLambda(index, index2, Length/XStep-1, Width/YStep-1, Length/XStep-1, Width/YStep-2, parameter)*(slice[Width/YStep-1][Length/XStep-1]-slice[Width/YStep-2][Length/XStep-1])/(stdYStep*(getEy(Width/YStep-2)+getEy(Width/YStep-1))) +
		parameter.GetQ(Length/XStep-1, Width/YStep-1, z)/(2*stdYStep) +
		parameter.GetQ(Length/XStep-1, Width/YStep-1, z)/(2*stdXStep)
	deltaHrt = deltaHrt * (2 * deltaT / parameter.Density[index])
	//fmt.Println(
	//	getLambda(index, index1, Length/XStep-1, Width/YStep-1, Length/XStep-2, Width/YStep-1, parameter)*(slice[Width/YStep-1][Length/XStep-1]-slice[Width/YStep-1][Length/XStep-2])/(stdXStep*(getEx(Length/XStep-2)+getEx(Length/XStep-1))),
	//	getLambda(index, index2, Length/XStep-1, Width/YStep-1, Length/XStep-1, Width/YStep-2, parameter)*(slice[Width/YStep-1][Length/XStep-1]-slice[Width/YStep-2][Length/XStep-1])/(stdYStep*(getEy(Width/YStep-2)+getEy(Width/YStep-1))),
	//	parameter.GetQ(slice[Width/YStep-1][Length/XStep-1])/(2*stdYStep),
	//	parameter.GetQ(slice[Width/YStep-1][Length/XStep-1])/(2*stdXStep),
	//	"right top",
	//)
	targetTemp := parameter.Enthalpy2Temp(parameter.Temp2Enthalpy(slice[Width/YStep-1][Length/XStep-1]) - deltaHrt)
	if c.alternating { // 需要修改焓的变化到温度变化的映射关系)
		c.thermalField1.Set(z, Width/YStep-1, Length/XStep-1, targetTemp, parameter.TemperatureBottom)
	} else {
		c.thermalField.Set(z, Width/YStep-1, Length/XStep-1, targetTemp, parameter.TemperatureBottom)
	}
}

// 计算右表面点的温度变化
func (c *calculatorWithArrDeque) calculatePointRA(deltaT float32, y, z int, slice *model.ItemType, parameter *Parameter) {
	var index = int(slice[y][Length/XStep-1]) - 1
	var index1 = int(slice[y][Length/XStep-2]) - 1
	var index2 = int(slice[y-1][Length/XStep-1]) - 1
	var index3 = int(slice[y+1][Length/XStep-1]) - 1

	var deltaHra = getLambda(index, index1, Length/XStep-1, y, Length/XStep-2, y, parameter)*(slice[y][Length/XStep-1]-slice[y][Length/XStep-2])/(stdXStep*(getEx(Length/XStep-2)+getEx(Length/XStep-1))) +
		getLambda(index, index2, Length/XStep-1, y, Length/XStep-1, y-1, parameter)*(slice[y][Length/XStep-1]-slice[y-1][Length/XStep-1])/(stdYStep*(getEy(y-1)+getEy(y))) +
		getLambda(index, index3, Length/XStep-1, y, Length/XStep-1, y+1, parameter)*(slice[y][Length/XStep-1]-slice[y+1][Length/XStep-1])/(stdYStep*(getEy(y+1)+getEy(y))) +
		parameter.GetQ(Length/XStep-1, y, z)/(2*stdXStep)
	deltaHra = deltaHra * (2 * deltaT / parameter.Density[index])
	//fmt.Println(
	//	getLambda(index, index1, Length/XStep-1, y, Length/XStep-2, y, parameter)*(slice[y][Length/XStep-1]-slice[y][Length/XStep-2])/(stdXStep*(getEx(Length/XStep-2)+getEx(Length/XStep-1))),
	//	getLambda(index, index2, Length/XStep-1, y, Length/XStep-1, y-1, parameter)*(slice[y][Length/XStep-1]-slice[y-1][Length/XStep-1])/(stdYStep*(getEy(y-1)+getEy(y))),
	//	getLambda(index, index3, Length/XStep-1, y, Length/XStep-1, y+1, parameter)*(slice[y][Length/XStep-1]-slice[y+1][Length/XStep-1])/(stdYStep*(getEy(y+1)+getEy(y))),
	//	parameter.GetQ(slice[y][Length/XStep-1])/(2*stdXStep),
	//	"right",
	//)
	//fmt.Println("parameter.Enthalpy2Temp: ", parameter.Enthalpy2Temp, "deltaHrt:", deltaHra, "△t:", deltaHra*parameter.Enthalpy2Temp, "right")
	targetTemp := parameter.Enthalpy2Temp(parameter.Temp2Enthalpy(slice[y][Length/XStep-1]) - deltaHra)
	if c.alternating { // 需要修改焓的变化到温度变化的映射关系
		c.thermalField1.Set(z, y, Length/XStep-1, targetTemp, parameter.TemperatureBottom)
	} else {
		c.thermalField.Set(z, y, Length/XStep-1, targetTemp, parameter.TemperatureBottom)
	}
}

// 计算right bottom点的温度变化
func (c *calculatorWithArrDeque) calculatePointRB(deltaT float32, z int, slice *model.ItemType, parameter *Parameter) {
	var index = int(slice[0][Length/XStep-1]) - 1
	var index1 = int(slice[0][Length/XStep-2]) - 1
	var index2 = int(slice[1][Length/XStep-1]) - 1

	var deltaHrb = getLambda(index, index1, Length/XStep-1, 0, Length/XStep-2, 0, parameter)*(slice[0][Length/XStep-1]-slice[0][Length/XStep-2])/(stdXStep*(getEx(Length/XStep-2)+getEx(Length/XStep-1))) +
		getLambda(index, index2, Length/XStep-1, 0, Length/XStep-1, 1, parameter)*(slice[0][Length/XStep-1]-slice[1][Length/XStep-1])/(stdYStep*(getEy(1)+getEy(0))) +
		parameter.GetQ(Length/XStep-1, 0, z)/(2*stdXStep)
	deltaHrb = deltaHrb * (2 * deltaT / parameter.Density[index])

	//fmt.Println(
	//	getLambda(index, index1, Length/XStep-1, 0, Length/XStep-2, 0, parameter)*(slice[0][Length/XStep-1]-slice[0][Length/XStep-2])/(stdXStep*(getEx(Length/XStep-2)+getEx(Length/XStep-1))),
	//	getLambda(index, index2, Length/XStep-1, 0, Length/XStep-1, 1, parameter)*(slice[0][Length/XStep-1]-slice[1][Length/XStep-1])/(stdYStep*(getEy(1)+getEy(0))),
	//	parameter.GetQ(slice[0][Length/XStep-1])/(2*stdXStep),
	//	"right bottom",
	//)
	//fmt.Println("parameter.Enthalpy2Temp: ", parameter.Enthalpy2Temp, "deltaHrt:", deltaHrb, "△t:", deltaHrb*parameter.Enthalpy2Temp, "right bottom")
	targetTemp := parameter.Enthalpy2Temp(parameter.Temp2Enthalpy(slice[0][Length/XStep-1]) - deltaHrb)
	if c.alternating { // 需要修改焓的变化到温度变化的映射关系
		c.thermalField1.Set(z, 0, Length/XStep-1, targetTemp, parameter.TemperatureBottom)
	} else {
		c.thermalField.Set(z, 0, Length/XStep-1, targetTemp, parameter.TemperatureBottom)
	}
}

// 计算下表面点的温度变化
func (c *calculatorWithArrDeque) calculatePointBA(deltaT float32, x, z int, slice *model.ItemType, parameter *Parameter) {
	var index = int(slice[0][x]) - 1
	var index1 = int(slice[0][x-1]) - 1
	var index2 = int(slice[0][x+1]) - 1
	var index3 = int(slice[1][x]) - 1
	var deltaHba = getLambda(index, index1, x, 0, x-1, 0, parameter)*(slice[0][x]-slice[0][x-1])/(stdXStep*(getEx(x-1)+getEx(x))) +
		getLambda(index, index2, x, 0, x+1, 0, parameter)*(slice[0][x]-slice[0][x+1])/(stdXStep*(getEx(x+1)+getEx(x))) +
		getLambda(index, index3, x, 0, x, 1, parameter)*(slice[0][x]-slice[1][x])/(stdYStep*(getEy(1)+getEy(0)))
	deltaHba = deltaHba * (2 * deltaT / parameter.Density[index])
	//fmt.Println(
	//	getLambda(index, index1, x, 0, x-1, 0, parameter)*(slice[0][x]-slice[0][x-1])/(stdXStep*(getEx(x-1)+getEx(x))),
	//	getLambda(index, index2, x, 0, x+1, 0, parameter)*(slice[0][x]-slice[0][x+1])/(stdXStep*(getEx(x+1)+getEx(x))),
	//	getLambda(index, index3, x, 0, x, 1, parameter)*(slice[0][x]-slice[1][x])/(stdYStep*(getEy(1)+getEy(0))),
	//	"bottom",
	//)
	//fmt.Println("parameter.Enthalpy2Temp: ", parameter.Enthalpy2Temp, "deltaHrt:", deltaHba, "△t:", deltaHba*parameter.Enthalpy2Temp, "bottom")
	targetTemp := parameter.Enthalpy2Temp(parameter.Temp2Enthalpy(slice[0][x]) - deltaHba)
	if c.alternating { // 需要修改焓的变化到温度变化的映射关系)
		c.thermalField1.Set(z, 0, x, targetTemp, parameter.TemperatureBottom)
	} else {
		c.thermalField.Set(z, 0, x, targetTemp, parameter.TemperatureBottom)
	}
}

// 计算left bottom点的温度变化
func (c *calculatorWithArrDeque) calculatePointLB(deltaT float32, z int, slice *model.ItemType, parameter *Parameter) {
	var index = int(slice[0][0]) - 1
	var index1 = int(slice[0][1]) - 1
	var index2 = int(slice[1][0]) - 1
	var deltaHlb = getLambda(index, index1, 1, 0, 0, 0, parameter)*(slice[0][0]-slice[0][1])/(stdXStep*(getEx(0)+getEx(1))) +
		getLambda(index, index2, 0, 1, 0, 0, parameter)*(slice[0][0]-slice[1][0])/(stdYStep*(getEy(1)+getEy(0)))
	deltaHlb = deltaHlb * (2 * deltaT / parameter.Density[index])
	//fmt.Println(
	//	getLambda(index, index1, 1, 0, 0, 0, parameter)*(slice[0][0]-slice[0][1])/(stdXStep*(getEx(0)+getEx(1))),
	//	getLambda(index, index2, 0, 1, 0, 0, parameter)*(slice[0][0]-slice[1][0])/(stdYStep*(getEy(1)+getEy(0))),
	//	"left bottom",
	//)
	//fmt.Println("parameter.Enthalpy2Temp: ", parameter.Enthalpy2Temp, "deltaHrt:", deltaHlb, "△t:", deltaHlb*parameter.Enthalpy2Temp, "left bottom")
	targetTemp := parameter.Enthalpy2Temp(parameter.Temp2Enthalpy(slice[0][0]) - deltaHlb)
	if c.alternating { // 需要修改焓的变化到温度变化的映射关系)
		c.thermalField1.Set(z, 0, 0, targetTemp, parameter.TemperatureBottom)
	} else {
		c.thermalField.Set(z, 0, 0, targetTemp, parameter.TemperatureBottom)
	}
}

// 计算左表面点温度的变化
func (c *calculatorWithArrDeque) calculatePointLA(deltaT float32, y, z int, slice *model.ItemType, parameter *Parameter) {
	var index = int(slice[y][0]) - 1
	var index1 = int(slice[y][1]) - 1
	var index2 = int(slice[y-1][0]) - 1
	var index3 = int(slice[y+1][0]) - 1
	var deltaHla = getLambda(index, index1, 1, y, 0, y, parameter)*(slice[y][0]-slice[y][1])/(stdXStep*(getEx(0)+getEx(1))) +
		getLambda(index, index2, 0, y-1, 0, y, parameter)*(slice[y][0]-slice[y-1][0])/(stdYStep*(getEy(y)+getEy(y-1))) +
		getLambda(index, index3, 0, y+1, 0, y, parameter)*(slice[y][0]-slice[y+1][0])/(stdYStep*(getEy(y)+getEy(y+1)))
	deltaHla = deltaHla * (2 * deltaT / parameter.Density[index])
	//fmt.Println(
	//	getLambda(index, index1, 1, y, 0, y, parameter)*(slice[y][0]-slice[y][1])/(stdXStep*(getEx(0)+getEx(1))),
	//	getLambda(index, index2, 0, y-1, 0, y, parameter)*(slice[y][0]-slice[y-1][0])/(stdYStep*(getEy(y)+getEy(y-1))),
	//	getLambda(index, index3, 0, y+1, 0, y, parameter)*(slice[y][0]-slice[y+1][0])/(stdYStep*(getEy(y)+getEy(y+1))),
	//	"left",
	//)
	//fmt.Println("parameter.Enthalpy2Temp: ", parameter.Enthalpy2Temp, "deltaHrt:", deltaHla, "△t:", deltaHla*parameter.Enthalpy2Temp, "left")
	targetTemp := parameter.Enthalpy2Temp(parameter.Temp2Enthalpy(slice[y][0]) - deltaHla)
	if c.alternating { // 需要修改焓的变化到温度变化的映射关系)
		c.thermalField1.Set(z, y, 0, targetTemp, parameter.TemperatureBottom)
	} else {
		c.thermalField.Set(z, y, 0, targetTemp, parameter.TemperatureBottom)
	}
}

// 计算内部点的温度变化
func (c *calculatorWithArrDeque) calculatePointIN(deltaT float32, x, y, z int, slice *model.ItemType, parameter *Parameter) {
	var index = int(slice[y][x]) - 1
	var index1 = int(slice[y][x-1]) - 1
	var index2 = int(slice[y][x+1]) - 1
	var index3 = int(slice[y-1][x]) - 1
	var index4 = int(slice[y+1][x]) - 1
	var deltaHin = getLambda(index, index1, x-1, y, x, y, parameter)*(slice[y][x]-slice[y][x-1])/(stdXStep*(getEx(x)+getEx(x-1))) +
		getLambda(index, index2, x+1, y, x, y, parameter)*(slice[y][x]-slice[y][x+1])/(stdXStep*(getEx(x)+getEx(x+1))) +
		getLambda(index, index3, x, y-1, x, y, parameter)*(slice[y][x]-slice[y-1][x])/(stdYStep*(getEy(y)+getEy(y-1))) +
		getLambda(index, index4, x, y+1, x, y, parameter)*(slice[y][x]-slice[y+1][x])/(stdYStep*(getEy(y)+getEy(y+1)))
	deltaHin = deltaHin * (2 * deltaT / parameter.Density[index])
	//fmt.Println("parameter.Enthalpy2Temp: ", parameter.Enthalpy2Temp, "deltaHrt:", deltaHin, "△t:", deltaHin*parameter.Enthalpy2Temp, "in")
	targetTemp := parameter.Enthalpy2Temp(parameter.Temp2Enthalpy(slice[y][x]) - deltaHin)
	if c.alternating { // 需要修改焓的变化到温度变化的映射关系)
		c.thermalField1.Set(z, y, x, targetTemp, parameter.TemperatureBottom)
	} else {
		c.thermalField.Set(z, y, x, targetTemp, parameter.TemperatureBottom)
	}
	//if x == Length / XStep - 4 && y == Width / YStep - 3 {
	//	fmt.Println(deltaHin, "in")
	//	fmt.Println(index, index1, index2, index3, index4)
	//	fmt.Println(
	//		getLambda(index, index1, x-1, y, x, y, parameter)*(slice[y][x]-slice[y][x-1])/(stdXStep*(getEx(x)+getEx(x-1))),
	//		getLambda(index, index2, x+1, y, x, y, parameter)*(slice[y][x]-slice[y][x+1])/(stdXStep*(getEx(x)+getEx(x+1))),
	//		getLambda(index, index3, x, y-1, x, y, parameter)*(slice[y][x]-slice[y-1][x])/(stdYStep*(getEy(y)+getEy(y-1))),
	//		getLambda(index, index4, x, y+1, x, y, parameter)*(slice[y][x]-slice[y+1][x])/(stdYStep*(getEy(y)+getEy(y+1))),
	//		parameter.Temp2Enthalpy(slice[y][x]),
	//		deltaHin,
	//		slice[y][x],
	//		parameter.Temp2Enthalpy(slice[y][x]) - deltaHin,
	//		targetTemp,
	//		"in",
	//	)
	//}
}

// 测试用
func NewCalculatorForGenerate() *calculatorWithArrDeque {
	c := &calculatorWithArrDeque{}
	// 初始化数据结构
	c.thermalField = deque.NewArrDeque(ZLength / ZStep)
	c.thermalField1 = deque.NewArrDeque(ZLength / ZStep)
	c.Field = c.thermalField
	return c
}

func (c *calculatorWithArrDeque) GenerateResult() *TemperatureFieldData {
	//zMax := ZLength / ZStep
	yMax := Width / YStep
	xMax := Length / XStep
	var minus float32
	initialTemp := float32(1600.0)
	var slice *model.ItemType
	var scale = float32(0.9865)
	var base1 = float32(1414.864)
	for i := 0; i < 4000; i++ {
		c.Field.AddFirst(initialTemp)
	}
	UpLength := int(UpLength)
	for i := UpLength - 1; i >= 0; i-- {
		slice = c.Field.GetSlice(i)
		// 从右向左减少
		for y := yMax - 1; y >= 0; y-- {
			minus = base1 - base1*float32(4000-1-i)/float32(4000-1)
			for x := xMax - 1; x >= 0; x-- {
				minus *= scale * scale
				slice[y][x] -= rand.Float32()*6.8 + minus
			}
		}

		// 从上到下减少
		for x := xMax - 1; x >= 0; x-- {
			minus = base1 - base1*float32(4000-1-i)/float32(4000-1)
			for y := yMax - 1; y >= 0; y-- {
				minus *= scale * scale
				slice[y][x] -= rand.Float32()*6.8 + minus
			}
		}
	}

	var sliceCopy *model.ItemType
	for i := UpLength; i < UpLength+30; i++ {
		slice = c.Field.GetSlice(i)
		sliceCopy = c.Field.GetSlice(UpLength - 1 - (i - UpLength))
		for i := 0; i < len(slice); i++ {
			for j := 0; j < len(slice[0]); j++ {
				slice[i][j] = sliceCopy[i][j]
			}
		}
	}

	for i := UpLength + 30; i < c.Field.Size(); i++ {
		slice = c.Field.GetSlice(i)
		for i := 0; i < len(slice); i++ {
			for j := 0; j < len(slice[0]); j++ {
				slice[i][j] = sliceCopy[i][j]
			}
		}
	}

	base2 := float32(414.864)
	scale2 := float32(0.9665)
	for i := 4000 - 1; i >= UpLength+30; i-- {
		slice = c.Field.GetSlice(i)
		// 从右向左减少
		for y := yMax - 1; y >= 0; y-- {
			minus = base2 - base2*float32(4000-(UpLength+30)-1-(i-(UpLength+30)))/float32(4000-1-(UpLength+30))
			for x := xMax - 1; x >= 0; x-- {
				minus *= scale2
				slice[y][x] -= rand.Float32()*5.8 + minus
			}
		}

		// 从上到下减少
		for x := xMax - 1; x >= 0; x-- {
			minus = base2 - base2*float32(4000-(UpLength+112)-1-(i-(UpLength+112)))/float32(4000-1-(UpLength+112))
			for y := yMax - 1; y >= 0; y-- {
				minus *= scale2
				slice[y][x] -= rand.Float32()*5.8 + minus
			}
		}
	}
	return c.BuildData()
}

func (c *calculatorWithArrDeque) GenerateResultForEncoder() *MiddleState {
	//zMax := ZLength / ZStep
	yMax := Width / YStep
	xMax := Length / XStep
	var minus float32
	initialTemp := float32(1600.0)
	var slice *model.ItemType
	var scale = float32(0.9865)
	var base1 = float32(1414.864)
	for i := 0; i < 4000; i++ {
		c.Field.AddFirst(initialTemp)
	}
	UpLength := int(UpLength)
	for i := UpLength - 1; i >= 0; i-- {
		slice = c.Field.GetSlice(i)
		// 从右向左减少
		for y := yMax - 1; y >= 0; y-- {
			minus = base1 - base1*float32(4000-1-i)/float32(4000-1)
			for x := xMax - 1; x >= 0; x-- {
				minus *= scale * scale
				slice[y][x] -= rand.Float32()*6.8 + minus
			}
		}

		// 从上到下减少
		for x := xMax - 1; x >= 0; x-- {
			minus = base1 - base1*float32(4000-1-i)/float32(4000-1)
			for y := yMax - 1; y >= 0; y-- {
				minus *= scale * scale
				slice[y][x] -= rand.Float32()*6.8 + minus
			}
		}
	}

	var sliceCopy *model.ItemType
	for i := UpLength; i < UpLength+30; i++ {
		slice = c.Field.GetSlice(i)
		sliceCopy = c.Field.GetSlice(UpLength - 1 - (i - UpLength))
		for i := 0; i < len(slice); i++ {
			for j := 0; j < len(slice[0]); j++ {
				slice[i][j] = sliceCopy[i][j]
			}
		}
	}

	for i := UpLength + 30; i < c.Field.Size(); i++ {
		slice = c.Field.GetSlice(i)
		for i := 0; i < len(slice); i++ {
			for j := 0; j < len(slice[0]); j++ {
				slice[i][j] = sliceCopy[i][j]
			}
		}
	}

	base2 := float32(414.864)
	scale2 := float32(0.9665)
	for i := 4000 - 1; i >= UpLength+30; i-- {
		slice = c.Field.GetSlice(i)
		// 从右向左减少
		for y := yMax - 1; y >= 0; y-- {
			minus = base2 - base2*float32(4000-(UpLength+30)-1-(i-(UpLength+30)))/float32(4000-1-(UpLength+30))
			for x := xMax - 1; x >= 0; x-- {
				minus *= scale2
				slice[y][x] -= rand.Float32()*5.8 + minus
			}
		}

		// 从上到下减少
		for x := xMax - 1; x >= 0; x-- {
			minus = base2 - base2*float32(4000-(UpLength+112)-1-(i-(UpLength+112)))/float32(4000-1-(UpLength+112))
			for y := yMax - 1; y >= 0; y-- {
				minus *= scale2
				slice[y][x] -= rand.Float32()*5.8 + minus
			}
		}
	}
	fmt.Println(Length/XStep, Width/YStep)
	topLength := (Length / XStep) * (Width / YStep)
	arcLength := (ZLength/ZStep - 2) * (Length/XStep + Width/YStep - 1)
	downLength := topLength
	fmt.Println(topLength, arcLength, downLength)
	res := &MiddleState{
		Top:    make([]int, topLength),
		Arc:    make([]int, arcLength),
		Bottom: make([]int, downLength),
	}
	index := 0
	var curSlice *model.ItemType
	alter := 1
	for i := 0; i < c.Field.Size(); i++ {
		if i == 0 {
			buildDataForEnd(res.Top, c.Field.GetSlice(i))
		} else if i == c.Field.Size()-1 {
			buildDataForEnd(res.Bottom, c.Field.GetSlice(i))
		} else {
			curSlice = c.Field.GetSlice(i)
			if alter == 1 {
				for j := 0; j < Length/XStep; j++ {
					res.Arc[index] = int(curSlice[Width/YStep-1][j])
					index++
				}
				for k := Width/YStep - 2; k >= 0; k-- {
					res.Arc[index] = int(curSlice[k][Length/XStep-1])
					index++
				}
			} else {
				for k := 0; k < Width/YStep; k++ {
					res.Arc[index] = int(curSlice[k][Length/XStep-1])
					index++
				}
				for j := Length/XStep - 2; j >= 0; j-- {
					res.Arc[index] = int(curSlice[Width/YStep-1][j])
					index++
				}
			}
			alter ^= 1
		}
	}

	return res
}

func buildDataForEnd(container []int, slice *model.ItemType) {
	index := 0
	left, right, bottom, top := 0, Length/XStep-1, 0, Width/YStep-1
	alter := 1
	for left < right && bottom < top {
		if alter == 1 {
			for c := left; c <= right; c++ {
				container[index] = int(slice[top][c])
				index++

			}
			for r := top - 1; r >= 0; r-- {
				container[index] = int(slice[r][right])
				index++
			}
		} else {
			for r := bottom; r <= top; r++ {
				container[index] = int(slice[r][right])
				index++
			}
			for c := right - 1; c >= 0; c-- {
				container[index] = int(slice[top][c])
				index++
			}
		}
		alter ^= 1
		right--
		top--
	}
	if left <= right {
		if alter == 1 {
			for c := left; c <= right; c++ {
				container[index] = int(slice[top][c])
				index++
			}
		} else {
			for c := right; c >= left; c-- {
				container[index] = int(slice[top][c])
				index++
			}
		}
	}
}

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
	Outer [][2]float32 `json:"outer"`
	Inner [][2]float32 `json:"inner"`
}

func (c *calculatorWithArrDeque) GenerateVerticalSlice1Data(index int) *VerticalSliceData1 {
	if index >= Width/YStep-1 {
		index = Width/YStep - 1
	}
	res := &VerticalSliceData1{
		Outer: make([][2]float32, 0),
		Inner: make([][2]float32, 0),
	}
	step := 0
	c.Field.Traverse(func(z int, item *model.ItemType) {
		step++
		if step == 5 {
			res.Outer = append(res.Outer, [2]float32{float32((z + 1) * model.ZStep), item[Width/YStep-1][Length/XStep-1-index]})
			res.Inner = append(res.Inner, [2]float32{float32((z + 1) * model.ZStep), item[0][Length/XStep-1-index]})
			step = 0
		}
	})
	return res
}

type VerticalSliceData2 struct {
	Length        int           `json:"length"`
	VerticalSlice [][84]float32 `json:"vertical_slice"`
	Solid         []int         `json:"solid"`
	Liquid        []int         `json:"liquid"`
	SolidJoin     Join          `json:"solid_join"`
	LiquidJoin    Join          `json:"liquid_join"`
}

type Join struct {
	IsJoin    bool `json:"is_join"`
	JoinIndex int  `json:"join_index"`
}

func (c *calculatorWithArrDeque) GenerateVerticalSlice2Data(index int) *VerticalSliceData2 {
	solidTemp := float32(1445.69)
	liquidTemp := float32(1506.77)
	scale := 5
	res := &VerticalSliceData2{
		Length:        c.Field.Size(),
		VerticalSlice: make([][84]float32, c.Field.Size()/scale),
		Solid:         make([]int, c.Field.Size()/scale),
		Liquid:        make([]int, c.Field.Size()/scale),
	}

	var temp float32
	var solidJoinSet, liquidJoinSet bool
	step := 0
	zIndex := 0
	c.Field.Traverse(func(z int, item *model.ItemType) {
		step++
		if step == scale {
			for i := 0; i < 42; i++ {
				res.VerticalSlice[zIndex][42+i] = item[i][Length/XStep-1-index]
			}
			for i := 42 - 1; i >= 0; i-- {
				res.VerticalSlice[zIndex][42-1-i] = item[i][Length/XStep-1-index]
			}
			for i := 0; i < 42; i++ {
				temp = item[i][Length/XStep-1-index]
				if temp <= solidTemp {
					res.Solid[zIndex] = 42 - i
					if res.Solid[zIndex] == 42 && !solidJoinSet {
						res.SolidJoin.IsJoin = true
						res.SolidJoin.JoinIndex = zIndex
						solidJoinSet = true
						fmt.Println(res.SolidJoin)
					}
					break
				} else {
					res.Solid[zIndex] = 0
				}
			}

			for i := 0; i < 42; i++ {
				temp = item[i][Length/XStep-1-index]
				if temp <= liquidTemp {
					res.Liquid[zIndex] = 42 - i
					if res.Liquid[zIndex] == 42 && !liquidJoinSet {
						res.LiquidJoin.IsJoin = true
						res.LiquidJoin.JoinIndex = zIndex
						liquidJoinSet = true
						fmt.Println(res.LiquidJoin)
					}
					break
				} else {
					res.Liquid[zIndex] = 0
				}
			}
			step = 0
			zIndex++
		}
	})
	fmt.Println(res)
	return res
}

// 测试用
func (c *calculatorWithArrDeque) Calculate() {
	for z := 0; z < 1; z++ {
		c.thermalField.AddFirst(c.castingMachine.CoolerConfig.StartTemperature)
		c.thermalField1.AddFirst(c.castingMachine.CoolerConfig.StartTemperature)
	}

	start := time.Now()
	for count := 0; count < 10; count++ {
		deltaT, _ := c.calculateTimeStep()
		c.calculateQ()
		c.calculateHeff()
		fmt.Println(c.steel1.Parameter.Q[0][:Length/XStep+Width/YStep])
		fmt.Println(c.steel1.Parameter.Heff[0][:Length/XStep+Width/YStep])
		cost := c.e.dispatchTask(deltaT, 0, c.Field.Size())

		if c.alternating {
			c.Field = c.thermalField1
		} else {
			c.Field = c.thermalField
		}

		for i := Width/YStep - 1; i > Width/YStep-6; i-- {
			for j := Length/XStep - 5; j <= Length/XStep-1; j++ {
				fmt.Printf("%.4f ", float64(c.Field.Get(c.Field.Size()-1, i, j)))
			}
			fmt.Print(i)
			fmt.Println()
		}
		c.alternating = !c.alternating
		fmt.Println("单此计算时间：", cost.Milliseconds())
	}

	fmt.Println("arr deque 总共消耗时间：", time.Since(start), "平均消耗时间: ", time.Since(start)/100)

	// 一个核心计算
	//c.CalculateSerially()
}

func (c *calculatorWithArrDeque) TestCalculateQ() {
	for z := 0; z < 1; z++ {
		c.thermalField.AddFirst(c.castingMachine.CoolerConfig.StartTemperature)
		c.thermalField1.AddFirst(c.castingMachine.CoolerConfig.StartTemperature)
	}

	c.calculateQOnline()
}
