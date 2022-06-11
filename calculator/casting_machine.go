package calculator

import (
	"encoding/json"
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
	Zone0 = 0 // 结晶区
)

var (
	OneSliceDuration time.Duration
)

type CastingMachine struct {
	Coordinate   model.Coordinate // 铸机的一些尺寸配置
	CoolerConfig model.CoolerCfg
}

func NewCastingMachine() *CastingMachine {
	// 根据 铸机 编号获取铸机配置
	castingMachine := CastingMachine{
		CoolerConfig: model.CoolerCfg{
			TargetTemperature: make(map[string]float32),
			V:                 int64(10 * 1.5 * 1000 / 60), // m / min，默认速度1.5
			SecondaryCoolingZoneCfg: model.SecondaryCoolingZoneCfg{
				SecondaryCoolingWaterCfg: make([]model.SecondaryCoolingWaterSection, 0),
				NozzleCfg: model.NozzleCfg{
					WideItems:   make([]model.WideItem, 0),
					NarrowItems: make([]model.NarrowItem, 0),
				},
				CoolingZoneCfg: make([]model.CoolingZone, 0),
			},
		},
	}
	return &castingMachine
}

func (c *CastingMachine) SetFromJson(coordinate model.Coordinate) {
	c.Coordinate = coordinate
	log.Info("铸机尺寸配置: ", c.Coordinate)
}

func (c *CastingMachine) SetCoolerConfig(env model.Env, nozzleCfgData []byte) {
	c.Coordinate.LevelHeight = env.LevelHeight
	c.CoolerConfig.StartTemperature = env.StartTemperature
	// 结晶器区
	c.CoolerConfig.NarrowSurfaceIn = env.Md.NarrowSurfaceIn
	c.CoolerConfig.NarrowSurfaceOut = env.Md.NarrowSurfaceOut
	c.CoolerConfig.NarrowWaterVolume = env.Md.NarrowSurfaceVolume
	c.CoolerConfig.WideSurfaceIn = env.Md.WideSurfaceIn
	c.CoolerConfig.WideSurfaceOut = env.Md.WideSurfaceOut
	c.CoolerConfig.WideWaterVolume = env.Md.WideSurfaceVolume
	// 二冷区
	c.CoolerConfig.SecondaryCoolingZoneCfg.SecondaryCoolingWaterCfg = env.SecondaryCoolingWaterCfg
	err := json.Unmarshal(nozzleCfgData, &c.CoolerConfig.SecondaryCoolingZoneCfg.NozzleCfg)
	if err != nil {
		log.Error("err:", err)
		return
	}
	var rollerNum int
	wideItems := c.CoolerConfig.SecondaryCoolingZoneCfg.NozzleCfg.WideItems
	for i := range env.CoolingZoneCfg {
		rollerNum = env.CoolingZoneCfg[i].End
		if rollerNum <= len(wideItems) {
			env.CoolingZoneCfg[i].EndDistance = wideItems[rollerNum-1].Distance
		} else {
			log.Error("err:", "辊子下标越界")
			return
		}
	}
	c.CoolerConfig.SecondaryCoolingZoneCfg.CoolingZoneCfg = env.CoolingZoneCfg
	log.WithFields(log.Fields{
		"StartTemperature":        env.StartTemperature,
		"NarrowSurfaceIn":         env.Md.NarrowSurfaceIn,
		"NarrowSurfaceOut":        env.Md.NarrowSurfaceOut,
		"NarrowWaterVolume":       env.Md.NarrowSurfaceVolume,
		"WideSurfaceIn":           env.Md.WideSurfaceIn,
		"WideSurfaceOut":          env.Md.WideSurfaceOut,
		"WideWaterVolume":         env.Md.WideSurfaceVolume,
		"SecondaryCoolingZoneCfg": c.CoolerConfig.SecondaryCoolingZoneCfg,
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
	if z <= c.Coordinate.MdLength {
		return Zone0
	}
	coolingZoneCfg := c.CoolerConfig.SecondaryCoolingZoneCfg.CoolingZoneCfg
	for i, v := range coolingZoneCfg {
		if float32(z) <= v.EndDistance {
			return i + 1
		}
	}
	return -1
}
