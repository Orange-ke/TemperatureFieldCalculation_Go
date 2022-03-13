package casting_machine

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
	XStep   = 5
	YStep   = 5
	ZStep   = 10
	Length  = 2700 / 2
	Width   = 420 / 2
	ZLength = 40000
)

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

// 元素类型
type ItemType [Width / YStep][Length / XStep]float32

type CastingMachine struct {
	Number                 int
	Name                   string
	CrystallizerLength     int         // 结晶器长度
	SecondaryCoolingConfig map[int]int // 二冷区冷却区尺寸的起始位置
	CoolerConfig           CoolerCfg
}
type CoolerCfg struct {
	Meniscus int // 弯月面高度

	StartTemperature float32 // 初始浇铸温度

	NarrowSurfaceIn        float32 // 窄面入水温度
	NarrowSurfaceOut       float32 // 窄面出水温度
	NarrowWaterConsumption float32 // 窄面水量

	WideSurfaceIn        float32 // 宽面入水温度
	WideSurfaceOut       float32 // 宽面出水温度
	WideWaterConsumption float32 // 宽面水量

	SprayTemperature      float32 // 二冷区喷淋水温度
	SprayWaterConsumption float32 // 水量
	SprayAirConsumption   float32 // 气量

	RollerWaterTemperature float32 // 棍子内冷水温度

	TargetTemperature map[string]float32 // 每个冷却区的目标温度

	V int64 // 拉速
}

func NewCastingMachine(number int) *CastingMachine {
	// 根据 铸机 编号获取铸机配置
	castingMachine := CastingMachine{
		Number:                 number,
		Name:                   "铸机1",
		CrystallizerLength:     100, // mm
		SecondaryCoolingConfig: make(map[int]int),
		CoolerConfig: CoolerCfg{
			TargetTemperature: make(map[string]float32),
			V:                 int64(10 * 1.5 * 1000 / 60), // m / min，默认速度1.5
		},
	}
	return &castingMachine
}

func (c *CastingMachine) SetCoolerConfig(env model.Env) {
	c.CoolerConfig.StartTemperature = env.StartTemperature
	c.CoolerConfig.NarrowSurfaceIn = env.NarrowSurfaceIn
	c.CoolerConfig.NarrowSurfaceOut = env.NarrowSurfaceOut
	c.CoolerConfig.WideSurfaceIn = env.WideSurfaceIn
	c.CoolerConfig.WideSurfaceOut = env.WideSurfaceOut
	c.CoolerConfig.SprayTemperature = env.SprayTemperature
	c.CoolerConfig.RollerWaterTemperature = env.RollerWaterTemperature
	log.WithFields(log.Fields{
		"StartTemperature": env.StartTemperature,
		"NarrowSurfaceIn": env.NarrowSurfaceIn,
		"NarrowSurfaceOut": env.NarrowSurfaceOut,
		"WideSurfaceIn": env.WideSurfaceIn,
		"WideSurfaceOut": env.WideSurfaceOut,
		"SprayTemperature": env.SprayTemperature,
		"RollerWaterTemperature": env.RollerWaterTemperature,
	}).Info("设置冷却参数")
}

func (c *CastingMachine) SetV(v float32) {
	c.CoolerConfig.V = int64(v * 1000 / 60)
	OneSliceDuration = time.Millisecond * time.Duration(1000*float32(ZStep)/float32(c.CoolerConfig.V)) // 10 / c.v
	log.WithFields(log.Fields{
		"V": c.CoolerConfig.V,
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

func (c *CastingMachine) SetSprayTemperature(sprayTemperature float32) {
	c.CoolerConfig.SprayTemperature = sprayTemperature
}

func (c *CastingMachine) SetRollerWaterTemperature(rollerWaterTemperature float32) {
	c.CoolerConfig.RollerWaterTemperature = rollerWaterTemperature
}

// 获取在那个冷却区
func (c *CastingMachine) WhichZone(z int) int {
	z = z * ZStep / StepZ // stepZ代表Z方向的缩放比例
	if z <= UpLength {    // upLength 代表结晶器的长度 R
		return Zone0
	} else {
		// todo 不同的区域返回不同的代号
		return Zone1
	}
}
