package calculator

import "lz/model"

// calculator 的接口定义

type Calculator interface {
	// 构建data
	BuildData() *TemperatureData
	// 构建slice data
	BuildSliceData(index int) *SlicePushDataStruct

	// 获取CalcHub
	GetCalcHub() *CalcHub

	// 初始化计算参数
	InitParameter(steelValue int)

	// 设置冷却器参数
	SetCoolerConfig(env model.Env)
	SetStartTemperature(startTemperature float32)
	SetNarrowSurfaceIn(narrowSurfaceIn float32)
	SetNarrowSurfaceOut(narrowSurfaceOut float32)
	SetWideSurfaceIn(wideSurfaceIn float32)
	SetWideSurfaceOut(wideSurfaceOut float32)
	SetSprayTemperature(sprayTemperature float32)
	SetRollerWaterTemperature(rollerWaterTemperature float32)
	SetV(v float32)

	// 运行
	Run()
	// 推送切片数据
	SliceDetailRun()

	// 设置拉尾坯
	SetStateTail()

	// 获取温度场数组的大小
	GetFieldSize() int
}
