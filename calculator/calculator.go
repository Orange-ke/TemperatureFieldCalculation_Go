package calculator

import "lz/casting_machine"

// calculator 的接口定义

type Calculator interface {
	// 构建data
	BuildData() *TemperatureFieldData
	// 构建slice data
	BuildSliceData(index int) *SlicePushDataStruct

	// 获取CalcHub
	GetCalcHub() *CalcHub

	// 初始化钢种
	InitSteel(steelValue int, castingMachine *casting_machine.CastingMachine)

	// 初始化铸机
	InitCastingMachine(castingMachineNumber int)

	// 获取钢种
	GetCastingMachine() *casting_machine.CastingMachine

	// 运行
	Run()

	// 设置拉尾坯
	SetStateTail()

	// 获取温度场数组的大小
	GetFieldSize() int

	GenerateResult() *TemperatureFieldData
}
