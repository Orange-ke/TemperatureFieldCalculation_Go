package steel_type

import (
	"fmt"
	"lz/casting_machine"
)

const (
	ArrayLength = 1600
	MinTemp     = 20
)

type Steel struct {
	Number         int
	Name           string
	Parameter      *Parameter
	CastingMachine *casting_machine.CastingMachine
}

type Parameter struct {
	Density  [ArrayLength]float32 // 密度
	Enthalpy [ArrayLength]float32 // 焓
	Lambda   [ArrayLength]float32 // 导热系数
	HEff     [ArrayLength]float32 // 综合换热系数
	Q        [ArrayLength]float32 // 热流密度
	C        [ArrayLength]float32 // 比热容
	GetHeff  func(T float32) float32
	GetQ     func(T float32) float32

	Enthalpy2Temp     float32
	TemperatureBottom float32 // 温度下限
}

func NewSteel(number int, castingMachine *casting_machine.CastingMachine) *Steel {
	// 根据钢种编号获取钢种信息
	// todo 根据 参数中的 钢种从 jmatpro 接口获取对应的物性参数
	// 1. 初始化网格划分的各个节点的初始温度
	fmt.Println("钢种编号:", number)

	parameter := Parameter{}
	steel := Steel{
		Number:         number,
		Name:           "钢种1",
		Parameter:      &parameter,
		CastingMachine: castingMachine,
	}

	// 2. 导热系数，200℃ 到 1600℃，随温度的上升而下降
	var LambdaStart = float32(25.0)
	var LambdaIter = float32(50.0-40.0) / (ArrayLength - MinTemp)
	for i := MinTemp - 1; i < (ArrayLength); i++ {
		steel.Parameter.Lambda[i] = LambdaStart + float32(i)*LambdaIter
	}

	// 3. 密度
	var DensityStart = float32(7.8) * 1000
	var DensityIter = float32(8.0-7.0) * 1000 / (ArrayLength - MinTemp)
	for i := MinTemp - 1; i < (ArrayLength); i++ {
		steel.Parameter.Density[i] = DensityStart - float32(i-MinTemp+1)*DensityIter
	}

	// 4. 焓值
	var EnthalpyStart = float32(11.0) * 1000
	var EnthalpyStep = float32(1320) * 1000 / (ArrayLength - MinTemp)
	for i := MinTemp - 1; i < (ArrayLength); i++ {
		steel.Parameter.Enthalpy[i] = EnthalpyStart + float32(i-MinTemp+1)*EnthalpyStep
	}

	// 5. 综合换热系数
	var HEffStart = float32(5.0) * 1e3
	var HEffStep = float32(20.0-15.0) * 1e3 / (ArrayLength - MinTemp)
	for i := MinTemp - 1; i < (ArrayLength); i++ {
		steel.Parameter.HEff[i] = HEffStart + float32(i-MinTemp+1)*HEffStep
	}

	// 6. 热流密度
	var QStart = float32(0.8) * 1e6 * 0.8
	var QStep = float32(1.65-0.8) * 1e6 / (ArrayLength - MinTemp) * 0.8
	for i := MinTemp - 1; i < (ArrayLength); i++ {
		steel.Parameter.Q[i] = QStart + float32(i-MinTemp+1)*QStep
	}

	// 7. 比热容
	var CStart = float32(0.44) * 1000
	var CStep = float32(0.82-0.44) * 1000 / (ArrayLength - MinTemp)
	for i := MinTemp - 1; i < (ArrayLength); i++ {
		steel.Parameter.C[i] = CStart + float32(i-MinTemp+1)*CStep
	}

	steel.Parameter.Enthalpy2Temp = (ArrayLength - MinTemp) / (steel.Parameter.Enthalpy[ArrayLength-1] - steel.Parameter.Enthalpy[MinTemp-1])
	return &steel
}

// 获取热流密度和综合换热系数
// s <= r
func (s *Steel) GetHeffLessThanR(T float32) float32 {
	//fmt.Println(s.Parameter.Q[int(T)], T-s.CastingMachine.CoolerConfig.NarrowSurfaceIn, "GetHeffLessThanR")
	// todo 待改
	divider := T - s.CastingMachine.CoolerConfig.NarrowSurfaceIn
	return s.Parameter.Q[int(T)] / divider
}

// s > r
func (s *Steel) GetHeffGreaterThanR(T float32) float32 {
	return s.Parameter.HEff[int(T)]
}

// s <= r
func (s *Steel) GetQLessThanR(T float32) float32 {
	return s.Parameter.Q[int(T)]
}

// s > r
func (s *Steel) GetQGreaterThanR(T float32) float32 {
	return s.Parameter.HEff[int(T)] * (T - s.CastingMachine.CoolerConfig.SprayTemperature)
}

// 获取不同冷却区对应的参数
func (s *Steel) SetParameter(z int) {
	if s.CastingMachine.WhichZone(z) == casting_machine.Zone0 {
		s.Parameter.GetHeff = s.GetHeffLessThanR
		s.Parameter.GetQ = s.GetQLessThanR
		s.Parameter.TemperatureBottom = s.CastingMachine.CoolerConfig.NarrowSurfaceIn
	} else if s.CastingMachine.WhichZone(z) == casting_machine.Zone1 {
		s.Parameter.GetHeff = s.GetHeffGreaterThanR
		s.Parameter.GetQ = s.GetQGreaterThanR
		s.Parameter.TemperatureBottom = s.CastingMachine.CoolerConfig.SprayTemperature
	}
}
