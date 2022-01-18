package calculator

import "time"

// calculator 的接口定义

type Calculator interface {
	// 构建data
	BuildData() *TemperatureData

	// 获取CalcHub
	GetCalcHub() *CalcHub

	// 计算时间步长，内部使用
	calculateTimeStep() (float32, time.Duration)

	// 并行计算
	calculateConcurrently(deltaT float32) time.Duration

	// 运行
	Run()
}
