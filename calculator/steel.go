package calculator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"lz/model"
	"sort"
)

const (
	ArrayLength = 1600
)

type Steel struct {
	Number                 int
	Name                   string
	LiquidPhaseTemperature float32
	SolidPhaseTemperature  float32
	Parameter              *Parameter
	CastingMachine         *CastingMachine
}

type Parameter struct {
	Density           [ArrayLength]float32           // 密度
	Enthalpy          [ArrayLength]float32           // 焓
	Lambda            [ArrayLength]float32           // 导热系数
	C                 [ArrayLength]float32           // 比热容
	Q                 [][model.WL]float32            // 热流密度
	Heff              [][model.WL]float32            // 综合换热系数
	GetHeff           func(x, y, z int) float32      // 获取综合换热系数
	GetQ              func(x, y, z int) float32      // 获取热流密度
	Enthalpy2Temp     func(enthalpy float32) float32 // 通过焓值获取对应的温度
	Temp2Enthalpy     func(temp float32) float32     // 通过温度获取焓值
	TemperatureBottom float32                        // 温度下限
}

func NewSteel(number int, castingMachine *CastingMachine) *Steel {
	// 根据钢种编号获取钢种信息
	// todo 根据 参数中的 钢种从 jmatpro 接口获取对应的物性参数
	// 1. 初始化网格划分的各个节点的初始温度
	fmt.Println("钢种编号:", number)
	// 获取固液相线温度
	phaseTemperatureData, err := ioutil.ReadFile("E:/GoWorkPlace/src/lz/conf/phase_temperature.json")
	if err != nil {
		log.Println("err", err)
		return nil
	}

	var phaseTemperature []model.PhaseTemperature
	err = json.Unmarshal(phaseTemperatureData, &phaseTemperature)
	if err != nil {
		log.Println("err", err)
		return nil
	}
	// 获取物性参数
	physicalParameterData, err := ioutil.ReadFile("E:/GoWorkPlace/src/lz/conf/physical_parameter.json")
	if err != nil {
		log.Println("err", err)
		return nil
	}
	var physicalParameter []model.PhysicalParameter
	err = json.Unmarshal(physicalParameterData, &physicalParameter)
	if err != nil {
		log.Println("err", err)
		return nil
	}
	sort.Slice(physicalParameter, func(i, j int) bool {
		return physicalParameter[i].Temperature < physicalParameter[j].Temperature
	})
	parameter := Parameter{
		Q:    make([][model.WL]float32, ZLength/ZStep),
		Heff: make([][model.WL]float32, ZLength/ZStep),
	}
	steel := Steel{
		Number:                 number,
		Name:                   phaseTemperature[0].SteelType.Name,
		LiquidPhaseTemperature: phaseTemperature[0].LiquidPhaseTemperature,
		SolidPhaseTemperature:  phaseTemperature[0].SolidPhaseTemperature,
		Parameter:              &parameter,
		CastingMachine:         castingMachine,
	}
	var t1, t2 int // 温度
	var step float32
	for i := 0; i < len(physicalParameter)-1; i++ {
		t1 = int(physicalParameter[i].Temperature)
		t2 = int(physicalParameter[i+1].Temperature)
		// 1. 导热系数
		steel.Parameter.Lambda[t1-1] = physicalParameter[i].ThermalConductivity
		steel.Parameter.Lambda[t2-1] = physicalParameter[i+1].ThermalConductivity
		step = (steel.Parameter.Lambda[t2-1] - steel.Parameter.Lambda[t1-1]) / 5
		for j := 1; j < t2-t1; j++ {
			steel.Parameter.Lambda[t1+j-1] = steel.Parameter.Lambda[t1-1] + step*float32(j)
		}
		// 2. 密度
		steel.Parameter.Density[t1-1] = physicalParameter[i].Density
		steel.Parameter.Density[t2-1] = physicalParameter[i+1].Density
		step = (steel.Parameter.Density[t2-1] - steel.Parameter.Density[t1-1]) / 5
		for j := 1; j < t2-t1; j++ {
			steel.Parameter.Density[t1+j-1] = steel.Parameter.Density[t1-1] + step*float32(j)
		}
		// 3. 焓值
		steel.Parameter.Enthalpy[t1-1] = physicalParameter[i].Enthalpy
		steel.Parameter.Enthalpy[t2-1] = physicalParameter[i+1].Enthalpy
		step = (steel.Parameter.Enthalpy[t2-1] - steel.Parameter.Enthalpy[t1-1]) / 5
		for j := 1; j < t2-t1; j++ {
			steel.Parameter.Enthalpy[t1+j-1] = steel.Parameter.Enthalpy[t1-1] + step*float32(j)
		}
		// 4. 比热容
		steel.Parameter.C[t1-1] = physicalParameter[i].SpecficHeat
		steel.Parameter.C[t2-1] = physicalParameter[i+1].SpecficHeat
		step = (steel.Parameter.C[t2-1] - steel.Parameter.C[t1-1]) / 5
		for j := 1; j < t2-t1; j++ {
			steel.Parameter.C[t1+j-1] = steel.Parameter.C[t1-1] + step*float32(j)
		}
	}
	// 5. 焓到温度的对应关系
	steel.Parameter.Enthalpy2Temp = func(enthalpy float32) float32 {
		left, right := 0, len(steel.Parameter.Enthalpy)-1
		for left < right {
			m := left + (right-left+1)>>1
			if steel.Parameter.Enthalpy[m] <= enthalpy {
				left = m
			} else {
				right = m - 1
			}
		}
		if enthalpy == steel.Parameter.Enthalpy[left] {
			return float32(left + 1)
		}
		if left+1 >= 1600 {
			left = 1598
		}
		//fmt.Println(steel.Parameter.Enthalpy, enthalpy)
		//fmt.Println(left, steel.Parameter.Enthalpy[left+1]-steel.Parameter.Enthalpy[left], enthalpy-steel.Parameter.Enthalpy[left])
		return float32(left+1) + 1/(steel.Parameter.Enthalpy[left+1]-steel.Parameter.Enthalpy[left])*(enthalpy-steel.Parameter.Enthalpy[left])
	}
	// 6. 温度到焓的对应关系
	steel.Parameter.Temp2Enthalpy = func(temp float32) float32 {
		t := int(temp) - 1
		if temp-1 == float32(t) {
			return steel.Parameter.Enthalpy[t]
		}
		return steel.Parameter.Enthalpy[t] + (steel.Parameter.Enthalpy[t+1]-steel.Parameter.Enthalpy[t])*(temp-float32(t)-1)
	}
	// 设置获取热流密度和综合换热系数函数
	steel.Parameter.GetHeff = func(x, y, z int) float32 {
		if x == Length/XStep-1 {
			return steel.Parameter.Heff[z][x+Width/YStep-y]
		} else {
			return steel.Parameter.Heff[z][x]
		}
	}
	steel.Parameter.GetQ = func(x, y, z int) float32 {
		if x == Length/XStep-1 {
			return steel.Parameter.Q[z][x+Width/YStep-y]
		} else {
			return steel.Parameter.Q[z][x]
		}
	}
	return &steel
}

// 获取不同冷却区对应的参数
func (s *Steel) SetParameter(z int) {
	zone := s.CastingMachine.WhichZone(z)
	if zone == -1 {
		log.Fatal("err: ", "冷却区超过范围")
		return
	}
	if zone == Zone0 {
		s.Parameter.TemperatureBottom = s.CastingMachine.CoolerConfig.NarrowSurfaceIn
	} else {
		s.Parameter.TemperatureBottom = s.CastingMachine.CoolerConfig.SecondaryCoolingZoneCfg.SecondaryCoolingWaterCfg[zone-1].SprayWaterTemperature
	}
}
