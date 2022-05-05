package calculator

import (
	log "github.com/sirupsen/logrus"
	"lz/model"
	"time"
)

// 铸机的规格 + 冷却器参数配置

// 步长设定
// 1. x方向（宽边方向）5mm
// 2. y方向（窄边方向）5mm
// 3. z方向（拉坯方向），板坯切片厚度方向5mm / 10mm

// 参数解释
// 1. 从节点[i, j] 到 [x, y] 实际等效导热系数 lambda
// 2. 每个节点的密度，density
// 3. 边界节点热流密度，Q
// 4. 边界节点综合换热系数，heff

// 全局变量
// 步长 和 铸坯的尺寸，单位mm

const (
	UpLength   = 500
	DownLength = 500
	ArcLength  = 3000

	StepX = 2
	StepY = 2
	StepZ = 10

	Zone0 = 0 // 结晶区
	Zone1 = 1 // 二冷区
)

var (
	OneSliceDuration time.Duration
)

type CastingMachine struct {
	MdLength               int         // 结晶器长度
	SecondaryCoolingConfig map[int]int // 二冷区冷却区尺寸的起始位置
	CoolerConfig           CoolerCfg
}
type CoolerCfg struct {
	LevelHeight int // 弯月面高度

	StartTemperature float32 // 初始浇铸温度

	NarrowSurfaceIn   float32 // 窄面入水温度
	NarrowSurfaceOut  float32 // 窄面出水温度
	NarrowWaterVolume float32 // 窄面水量

	WideSurfaceIn   float32 // 宽面入水温度
	WideSurfaceOut  float32 // 宽面出水温度
	WideWaterVolume float32 // 宽面水量

	// todo
	TargetTemperature map[string]float32 // 每个冷却区的目标温度

	V int64 // 拉速
}

func NewCastingMachine() *CastingMachine {
	// 根据 铸机 编号获取铸机配置
	castingMachine := CastingMachine{
		SecondaryCoolingConfig: make(map[int]int),
		CoolerConfig: CoolerCfg{
			TargetTemperature: make(map[string]float32),
			V:                 int64(10 * 1.5 * 1000 / 60), // m / min，默认速度1.5
		},
	}
	return &castingMachine
}

func (c *CastingMachine) SetFromJson(coordinate model.Coordinate) {
	c.MdLength = coordinate.MdLength
	log.Info("结晶器长度为: ", c.MdLength)
}

func (c *CastingMachine) SetCoolerConfig(env model.Env) {
	c.CoolerConfig.StartTemperature = env.StartTemperature
	c.CoolerConfig.NarrowSurfaceIn = env.Md.NarrowSurfaceIn
	c.CoolerConfig.NarrowSurfaceOut = env.Md.NarrowSurfaceOut
	c.CoolerConfig.NarrowWaterVolume = env.Md.NarrowSurfaceVolume
	c.CoolerConfig.WideSurfaceIn = env.Md.WideSurfaceIn
	c.CoolerConfig.WideSurfaceOut = env.Md.WideSurfaceOut
	c.CoolerConfig.WideWaterVolume = env.Md.WideSurfaceVolume
	log.WithFields(log.Fields{
		"StartTemperature":  env.StartTemperature,
		"NarrowSurfaceIn":   env.Md.NarrowSurfaceIn,
		"NarrowSurfaceOut":  env.Md.NarrowSurfaceOut,
		"NarrowWaterVolume": env.Md.NarrowSurfaceVolume,
		"WideSurfaceIn":     env.Md.WideSurfaceIn,
		"WideSurfaceOut":    env.Md.WideSurfaceOut,
		"WideWaterVolume":   env.Md.WideSurfaceVolume,
		// todo
	}).Info("设置冷却参数")
}

func (c *CastingMachine) SetV(v float32) {
	c.CoolerConfig.V = int64(v * 1000 / 60)
	OneSliceDuration = time.Millisecond * time.Duration(1000*float32(model.ZStep)/float32(c.CoolerConfig.V)) // 10 / c.v
	log.WithFields(log.Fields{
		"V":                c.CoolerConfig.V,
		"oneSliceDuration": OneSliceDuration.Milliseconds(),
	}).Info("设置拉速")
}

// 冷却器参数单独设置
func (c *CastingMachine) SetStartTemperature(startTemperature float32) {
	c.CoolerConfig.StartTemperature = startTemperature
}

func (c *CastingMachine) SetNarrowSurfaceIn(narrowSurfaceIn float32) {
	c.CoolerConfig.NarrowSurfaceIn = narrowSurfaceIn
}

func (c *CastingMachine) SetNarrowSurfaceOut(narrowSurfaceOut float32) {
	c.CoolerConfig.NarrowSurfaceOut = narrowSurfaceOut
}

func (c *CastingMachine) SetWideSurfaceIn(wideSurfaceIn float32) {
	c.CoolerConfig.WideSurfaceIn = wideSurfaceIn
}

func (c *CastingMachine) SetWideSurfaceOut(wideSurfaceOut float32) {
	c.CoolerConfig.WideSurfaceOut = wideSurfaceOut
}

// 获取在那个冷却区
func (c *CastingMachine) WhichZone(z int) int {
	z = z * model.ZStep / StepZ // stepZ代表Z方向的缩放比例
	if z <= UpLength {          // upLength 代表结晶器的长度 R
		return Zone0
	} else {
		// todo 不同的区域返回不同的代号
		return Zone1
	}
}
