package calculator

// calculator 的接口定义

type Calculator interface {
	// 构建data
	BuildData() *TemperatureData

	// 获取CalcHub
	GetCalcHub() *CalcHub

	// 运行
	Run()
}
