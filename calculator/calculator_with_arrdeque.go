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
		c.e = newExecutorBaseOnSlice(6)
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
	up := coordinate.CenterStartDistance
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
	}, 0, c.Field.Size())
	if min >= 0.4 {
		min = 0.4
	}
	fmt.Println("计算deltaT花费的时间：", time.Since(start).Milliseconds(), min)
	return min, time.Since(start)
}

// 离线计算计算热流密度: 暂时未用到
func (c *calculatorWithArrDeque) calculateQOffline() {
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
		}, 0, (c.castingMachine.Coordinate.MdLength-int(c.castingMachine.Coordinate.LevelHeight))/ZStep)
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
			if item[0][j] > c.steel1.LiquidPhaseTemperature {
				c.steel1.Parameter.Q[z][j] = initialQ
				wideSurfaceEnergy += c.steel1.Parameter.Q[z][j] * float32(XStep*ZStep) / 1e6
			} else {
				break
			}
		}
		start := j - 1
		for ; j < Length/XStep; j++ {
			c.steel1.Parameter.Q[z][j] = initialQ - initialQ*0.7*(float32((j-start)*XStep)-float32(XStep)/2)/float32((Length/XStep-1-start)*XStep)
			wideSurfaceEnergy += c.steel1.Parameter.Q[z][j] * float32(XStep*ZStep) / 1e6
		}
	}, 0, (c.castingMachine.Coordinate.MdLength-int(c.castingMachine.Coordinate.LevelHeight))/ZStep)
	return wideSurfaceEnergy
}

func (c *calculatorWithArrDeque) calculateNarrowSurfaceEnergy(narrowSurfaceH float32) float32 {
	var narrowSurfaceEnergy float32
	var initialQ float32
	c.Field.Traverse(func(z int, item *model.ItemType) {
		initialQ = 1 / (ROfWater() + ROfCu() + 1/narrowSurfaceH) * (item[0][Length/XStep-1] - c.castingMachine.CoolerConfig.NarrowSurfaceIn)
		i := 0
		for ; i < Width/YStep; i++ {
			if item[i][0] > c.steel1.LiquidPhaseTemperature {
				c.steel1.Parameter.Q[z][Length/XStep+Width/YStep-1-i] = initialQ
				narrowSurfaceEnergy += c.steel1.Parameter.Q[z][Length/XStep+Width/YStep-1-i] * float32(YStep*ZStep) / 1e6
			} else {
				break
			}
		}
		start := i - 1
		for ; i < Width/YStep; i++ {
			c.steel1.Parameter.Q[z][Length/XStep+Width/YStep-1-i] = initialQ - (initialQ * 0.7 * (float32((i-start)*YStep) - float32(YStep)/2) / float32((Width/YStep-1-start)*YStep))
			narrowSurfaceEnergy += c.steel1.Parameter.Q[z][Length/XStep+Width/YStep-1-i] * float32(YStep*ZStep) / 1e6
		}
	}, 0, (c.castingMachine.Coordinate.MdLength-int(c.castingMachine.Coordinate.LevelHeight))/ZStep)
	return narrowSurfaceEnergy
}

// 在线计算热流密度和综合换热系数
func (c *calculatorWithArrDeque) calculateQAndHeffOnline() {
	// 结晶器先计算热流密度Q再计算综合换热系数Heff
	c.calculateQOnlineAtMd()
	c.calculateHeffOnlineAtMd()
	if c.Field.Size() > (c.castingMachine.Coordinate.MdLength-int(c.castingMachine.Coordinate.LevelHeight))/ZStep {
		// 二冷区先计算平均综合换热系数再计算热流密度
		c.calculateHeffOnlineAtSecondaryCoolingZone()
		c.calculateQOnlineAtSecondaryCoolingZone()
	}
}

func (c *calculatorWithArrDeque) calculateQOnlineAtMd() {
	start := time.Now()
	energyScale := float32(c.Field.Size()) / (float32(c.castingMachine.Coordinate.MdLength) / float32(ZStep))
	if energyScale >= (float32(c.castingMachine.Coordinate.MdLength)-c.castingMachine.Coordinate.LevelHeight)/float32(c.castingMachine.Coordinate.MdLength) {
		energyScale = (float32(c.castingMachine.Coordinate.MdLength) - c.castingMachine.Coordinate.LevelHeight) / float32(c.castingMachine.Coordinate.MdLength)
	}
	fmt.Println("energyScale: ", energyScale)
	left, right := float32(500.0), float32(5000.0) // 二分的上下界
	var wideSurfaceH float32
	err := float32(0.000001)
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
	fmt.Println("计算结晶器热流密度所需时间：", time.Since(start).Milliseconds())
}

// 根据二冷区冷区参数计算对应的综合换热系数
func (c *calculatorWithArrDeque) calculateHeffOnlineAtSecondaryCoolingZone() {
	// Hi、Hi_1 代表辊子距离结晶器液面高度
	// D0 cm 铸坯厚度
	// Lwi 外弧线上辊距 cm
	// Ri 连铸机圆弧主半径 cm
	// v 拉速
	// Hi 第i个辊子处距结晶器液面的垂直高度
	// Hi_1 第i-1个辊子处距结晶器液面的垂直高度
	// Si_1 前一个辊子处坯壳厚度
	// Tm 凝固温度，用液相线温度
	// Tma 铸坯坯壳平均温度
	// 宽面，窄面分开计算
	wideItems := c.castingMachine.CoolerConfig.SecondaryCoolingZoneCfg.NozzleCfg.WideItems
	L := float32(c.castingMachine.Coordinate.Length) / 10 // 铸坯宽面尺寸
	coolingZoneCfg := c.castingMachine.CoolerConfig.SecondaryCoolingZoneCfg.CoolingZoneCfg
	cooingWaterCfg := c.castingMachine.CoolerConfig.SecondaryCoolingZoneCfg.SecondaryCoolingWaterCfg
	var envTemp = 70.0
	var AB, BC, CD, DE, Ds, sprayWidth, Hbr float32
	var Deformation, centerRollersDistance, v, Si_1, Tm, Tma, S, Volume, T, R0, Ts_ float64
	var preDistance = float32(c.castingMachine.Coordinate.MdLength) - c.castingMachine.Coordinate.LevelHeight
	var curDistance float32
	var startSliceIndex, endSliceIndex int
	// 计算宽面
	for _, item := range wideItems {
		if c.Field.Size() < int(preDistance)/ZStep {
			break
		}
		// step1. 计算平均综合换热系数
		Ds = item.CenterSpraySection.Thickness // 喷淋厚度
		centerRollersDistance = float64(item.RollerDistance / 10.0)
		AB = (L - Ds) / 2.0
		v = float64(c.castingMachine.CoolerConfig.V) / 10.0 * 60.0                                             // 拉速 mm/s -> cm/min
		Si_1 = float64(c.calculateSolidThickness(preDistance, "Wide"))                                         // 计算当前辊子处对应的坯壳厚度
		Tm = float64(c.steel1.LiquidPhaseTemperature)                                                          // 液相线温度
		Tma = float64(c.calculateTma(preDistance, "Wide"))                                                     // 坯壳平均温度
		Deformation = calculateDeformation(centerRollersDistance, v, float64(item.Distance/10), Si_1, Tm, Tma) // 计算鼓肚量
		DE = calculateDE(float64(item.InnerDiameter/10), float64(item.OuterDiameter/10), Deformation)          // 计算辊子直接接触宽度
		BC = Ds
		CD = AB - DE
		sprayWidth = min(item.CenterSpraySection.RightLimit-item.CenterSpraySection.LeftLimit, float32(c.castingMachine.Coordinate.Length)) // 喷淋宽度
		Ts_ = float64(c.calculateTs(preDistance, "Wide"))                                                                                   // 辊子对应铸坯表面平均温度
		Hbr = calculateHbr(Ts_, envTemp)                                                                                                    // 计算空气换热系数
		S = float64(sprayWidth*Ds) / 1e6                                                                                                    // 喷淋面积
		Volume = float64(cooingWaterCfg[item.CoolingZone-1].InnerArcWaterVolume / float32(coolingZoneCfg[item.CoolingZone-1].End-coolingZoneCfg[item.CoolingZone-1].Start+1) / 60.0)
		R0 = float64(item.InnerDiameter) / 2.0 / 10.0         // 辊子半径
		T = float64(c.calculateT(preDistance, item.Distance)) // 计算喷淋区域平均温度
		// step2. 确定辊间距对应影响的切片范围，然后更新
		curDistance = item.Distance
		startSliceIndex = int(preDistance / float32(ZStep))
		endSliceIndex = int(curDistance / float32(ZStep))
		preDistance = curDistance
		if cooingWaterCfg[item.CoolingZone-1].InnerArcWaterVolume == 0.0 {
			for z := startSliceIndex; z < endSliceIndex; z++ {
				for j := 0; j < Length/XStep; j++ {
					c.steel1.Parameter.Heff[z][j] = Hbr
				}
				continue
			}
		}
		heff := calculateAverageHeffHelper(L, AB, BC, CD, DE, Hbr, item.Medium, S, Volume, T, float64(Ds), R0, Ts_) // 计算平均综合换热系数
		//log.Info("平均综合换热系数：", heff)
		for z := startSliceIndex; z < endSliceIndex; z++ {
			for j := 0; j < int(sprayWidth/2)/XStep; j++ {
				c.steel1.Parameter.Heff[z][j] = heff
			}
			for j := int(sprayWidth/2) / XStep; j < Length/XStep; j++ {
				c.steel1.Parameter.Heff[z][j] = Hbr
			}
		}
	}
	// 计算窄面, 逻辑比较相似，但是有些不一样，因此还是分开处理
	// 如果分区存在喷淋冷却则按照宽面的思路计算，否则，只计算空冷
	narrowItems := c.castingMachine.CoolerConfig.SecondaryCoolingZoneCfg.NozzleCfg.NarrowItems
	if len(narrowItems) <= 0 {
		return
	}
	W := float32(c.castingMachine.Coordinate.Width) / 10 // 铸坯宽面尺寸
	preDistance = float32(c.castingMachine.Coordinate.MdLength) - c.castingMachine.Coordinate.LevelHeight
	startSliceIndex = 0
	endSliceIndex = 0
	for _, item := range narrowItems {
		if c.Field.Size() < int(preDistance)/ZStep {
			break
		}
		// step1. 计算平均综合换热系数
		Ds = item.SpraySection1.Thickness // 喷淋厚度
		centerRollersDistance = float64(item.RollerDistance / 10.0)
		AB = (W - Ds) / 2.0
		v = float64(c.castingMachine.CoolerConfig.V) / 10.0 * 60.0                                                                 // 拉速 mm/s -> cm/min
		Si_1 = float64(c.calculateSolidThickness(preDistance, "Narrow"))                                                           // 计算当前辊子处对应的坯壳厚度
		Tm = float64(c.steel1.LiquidPhaseTemperature)                                                                              // 液相线温度
		Tma = float64(c.calculateTma(preDistance, "Narrow"))                                                                       // 坯壳平均温度
		Deformation = calculateDeformation(centerRollersDistance, v, float64((preDistance+item.RollerDistance)/10), Si_1, Tm, Tma) // 计算鼓肚量
		DE = calculateDE(float64(item.Diameter/10), float64(item.Diameter/10), Deformation)                                        // 计算辊子直接接触宽度
		BC = Ds
		CD = AB - DE
		sprayWidth = min(item.SpraySection1.Width, float32(c.castingMachine.Coordinate.Width)) // 喷淋宽度
		Ts_ = float64(c.calculateTs(preDistance, "Narrow"))                                    // 辊子对应铸坯表面平均温度
		Hbr = calculateHbr(Ts_, envTemp)                                                       // 计算空气换热系数
		S = float64(sprayWidth*Ds) / 1e6                                                       // 喷淋面积
		Volume = float64(cooingWaterCfg[item.CoolingZone-1].NarrowSideWaterVolume / float32(len(narrowItems)) / 60.0)
		R0 = float64(item.Diameter) / 2.0 / 10.0                                // 辊子半径
		T = float64(c.calculateT(preDistance, preDistance+item.RollerDistance)) // 计算喷淋区域平均温度
		// step2. 确定辊间距对应影响的切片范围，然后更新
		curDistance = preDistance + item.RollerDistance
		startSliceIndex = int(preDistance / float32(ZStep))
		endSliceIndex = int(curDistance / float32(ZStep))
		preDistance = curDistance
		heff := calculateAverageHeffHelper(L, AB, BC, CD, DE, Hbr, Water, S, Volume, T, float64(Ds), R0, Ts_) // 计算平均综合换热系数
		log.Info("平均综合换热系数：", heff, " sprayWidth: ", sprayWidth, " Width: ", Width)
		for z := startSliceIndex; z <= endSliceIndex; z++ {
			for i := 0; i < int(sprayWidth/2)/YStep; i++ {
				c.steel1.Parameter.Heff[z][Length/XStep+i] = heff
			}
			//for i := int(sprayWidth/2) / YStep; i < Width/YStep; i++ {
			//	c.steel1.Parameter.Heff[z][Length/XStep+i] = Hbr
			//}
		}
	}
	startSliceIndex = int(preDistance / float32(ZStep))
	endSliceIndex = c.Field.Size()
	for z := startSliceIndex + 1; z < endSliceIndex; {
		Ts_ = float64(c.calculateTs(preDistance, "Narrow")) // 辊子对应铸坯表面平均温度
		Hbr = calculateHbr(Ts_, envTemp)                    // 计算空气换热系数
		for i := 0; i < Width/YStep; i++ {
			c.steel1.Parameter.Heff[z][Length/XStep+i] = Hbr
		}
		z++
		preDistance = float32(z * ZStep)
	}
}

func (c *calculatorWithArrDeque) calculateQOnlineAtSecondaryCoolingZone() {
	//start := time.Now()
	c.Field.Traverse(func(z int, item *model.ItemType) {
		for j := 0; j < Length/XStep; j++ {
			c.steel1.Parameter.Q[z][j] = c.steel1.Parameter.Heff[z][j] * (item[Width/YStep-1][j] - c.castingMachine.CoolerConfig.WideSurfaceIn)
		}
		for i := 0; i < Width/YStep; i++ {
			c.steel1.Parameter.Q[z][Length/XStep+i] = c.steel1.Parameter.Heff[z][Length/XStep+i] * (item[i][Length/XStep-1] - c.castingMachine.CoolerConfig.NarrowSurfaceIn)
		}
	}, (c.castingMachine.Coordinate.MdLength-int(c.castingMachine.Coordinate.LevelHeight))/ZStep, c.Field.Size())
	//fmt.Println("计算综合换热系数所需时间：", time.Since(start).Milliseconds())
}

// 计算辊子对应铸坯坯壳平均温度
func (c *calculatorWithArrDeque) calculateTma(distance float32, pos string) float32 {
	// 前一个辊子
	sliceIndex := int(distance/float32(ZStep)) - 1
	slice := c.Field.GetSlice(sliceIndex)
	var sum float32
	if pos == "Wide" {
		for i := 0; i < Length/XStep; i++ {
			sum += (c.steel1.LiquidPhaseTemperature + slice[Width/YStep-1][i]) / 2.0
		}
		return sum / float32(Length/XStep)
	} else {
		for i := 0; i < Width/YStep; i++ {
			sum += (c.steel1.LiquidPhaseTemperature + slice[i][Length/XStep-1]) / 2.0
		}
		return sum / float32(Width/YStep)
	}
}

// 计算辊子对应铸坯表面的平均温度
func (c *calculatorWithArrDeque) calculateTs(distance float32, pos string) float32 {
	// 前一个辊子
	sliceIndex := int(distance/float32(ZStep)) - 1
	slice := c.Field.GetSlice(sliceIndex)
	var sum float32
	if pos == "Wide" {
		for i := 0; i < Length/XStep; i++ {
			sum += slice[width/YStep-1][i]
		}
		return sum / float32(Length/XStep)
	} else {
		for i := 0; i < Width/YStep; i++ {
			sum += slice[i][Length/XStep-1]
		}
		return sum / float32(Width/YStep)
	}
}

// 计算喷淋区域平均温度
func (c *calculatorWithArrDeque) calculateT(preDistance, distance float32) float32 {
	startIndex := int(preDistance/float32(ZStep)) - 1
	endIndex := int(distance / float32(ZStep))
	var sum float32
	var count int
	c.Field.Traverse(func(z int, item *model.ItemType) {
		for j := 0; j < Length/XStep; j++ {
			sum += item[Width/XStep-1][j]
			count++
		}
	}, startIndex, endIndex)
	return sum / float32(count)
}

// 计算外弧辊距
func (c *calculatorWithArrDeque) calculateOuterRollersDistance(rollerDistance, distance float32) float32 {
	r := c.castingMachine.Coordinate.R
	halfWidth := float32(c.castingMachine.Coordinate.Width / 2)
	if distance > c.castingMachine.Coordinate.CenterStartDistance && distance < c.castingMachine.Coordinate.CenterEndDistance {
		return rollerDistance * r / (r - halfWidth)
	}
	return rollerDistance
}

// 计算平均坯壳厚度
func (c *calculatorWithArrDeque) calculateSolidThickness(distance float32, pos string) float32 {
	// 前一个辊子
	sliceIndex := int(distance/float32(ZStep)) - 1
	slice := c.Field.GetSlice(sliceIndex)
	liquidTemp := c.steel1.LiquidPhaseTemperature
	var sum, count float32
	if pos == "Wide" {
		for i := 0; i < Length/XStep; i++ {
			count = 0
			for j := Width/YStep - 1; j >= 0; j-- {
				if slice[j][i] <= liquidTemp {
					count++
				} else {
					break
				}
			}
			sum += count * float32(YStep)
		}
		//fmt.Println("calculateSolidThickness wide: ", sum, float32(Length/XStep))
		return sum / float32(Length/XStep)
	} else {
		for i := 0; i < Width/YStep; i++ {
			count = 0
			for j := Length/XStep - 1; j >= 0; j-- {
				if slice[i][j] <= liquidTemp {
					count++
				} else {
					break
				}
			}
			sum += count * float32(XStep)
		}
		//fmt.Println("calculateSolidThickness narrow: ", sum, float32(Width/YStep))
		return sum / float32(Width/YStep)
	}
}

// 计算综合换热系数
func (c *calculatorWithArrDeque) calculateHeffOnlineAtMd() {
	//start := time.Now()
	if c.runningState == stateRunning {
		c.Field.Traverse(func(z int, item *model.ItemType) {
			for j := 0; j < Length/XStep; j++ {
				c.steel1.Parameter.Heff[z][j] = c.steel1.Parameter.Q[z][j] / (item[Width/YStep-1][j] - c.castingMachine.CoolerConfig.WideSurfaceIn)
			}
			for i := 0; i < Width/YStep; i++ {
				c.steel1.Parameter.Heff[z][Length/XStep+i] = c.steel1.Parameter.Q[z][Length/XStep+i] / (item[i][Length/XStep-1] - c.castingMachine.CoolerConfig.NarrowSurfaceIn)
			}
		}, 0, (c.castingMachine.Coordinate.MdLength-int(c.castingMachine.Coordinate.LevelHeight))/ZStep)
	}
	//fmt.Println("计算综合换热系数所需时间：", time.Since(start).Milliseconds())
}

func (c *calculatorWithArrDeque) Run() {
	c.runningState = stateRunning
	// 先计算timeStep
	var duration, calcDuration, gap time.Duration
	var deltaT float32
LOOP:
	for {
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
				c.calculateQAndHeffOnline()
				//fmt.Println("Q: ", c.steel1.Parameter.Q[c.Field.Size()-1][:Length/XStep+Width/YStep])
				//fmt.Println("Heff: ", c.steel1.Parameter.Heff[c.Field.Size()-1][:Length/XStep+Width/YStep])
				//fmt.Println()
				//for i := 0; i < c.Field.Size(); i++ {
				//	fmt.Print(i, " ")
				//	for j := 0; j < Length/XStep+Width/YStep; j++{
				//		fmt.Print(c.steel1.Parameter.Heff[i][j], " ")
				//	}
				//	fmt.Println()
				//}
				deltaT, _ = c.calculateTimeStep()
				calcDuration = c.e.dispatchTask(deltaT, 0, c.Field.Size()) // c.ThermalField.Field 最开始赋值为 ThermalField对应的指针
				fmt.Println("计算单次时间：", calcDuration.Milliseconds(), "ms")
				gap = time.Duration(int64(deltaT*1e9)) - calcDuration
				if gap < 0 {
					gap = 0
				}
				duration += time.Duration(int64(deltaT * 1e9))
			}

			fmt.Println("时间步长: ", deltaT, gap, duration)
			// todo 加速计算过程
			//time.Sleep(gap)
			if c.alternating {
				c.Field = c.thermalField1
			} else {
				c.Field = c.thermalField
			}

			c.updateSliceInfo(time.Duration(int64(deltaT * 1e9)))
			//if !c.Field.IsEmpty() {
			//	for i := Width/YStep - 1; i >= 0; i-- {
			//		for j := 0; j <= Length/XStep-1; j++ {
			//			fmt.Printf("%.2f ", c.Field.Get(c.Field.Size()-1, i, j))
			//		}
			//		fmt.Println()
			//	}
			//}
			c.alternating = !c.alternating // 仅在这里修改
			log.WithFields(log.Fields{"deltaT": deltaT, "cost": duration.Milliseconds()}).Info("计算一次")
			if duration > time.Second*4 {
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
		log.Info("updateSliceInfo: 切片已满")
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
		parameter.GetQ(Length/XStep-1, Width/YStep, z)/(2*stdYStep) +
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
	//fmt.Println("deltaHrt:", deltaHra, "Q:", parameter.GetQ(Length/XStep-1, y, z)/(2*stdXStep), "right")
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
