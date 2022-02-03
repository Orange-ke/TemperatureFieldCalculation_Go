package calculator

import (
	"fmt"
)

// 初始化计算参数
func initParameters(steelValue int, parameter *parameter) {
	// 方程初始条件为 T = Tw，Tw为钢水刚到弯月面处的温度。

	// 对于1/4模型，如果不考虑沿着拉坯方向的传热，则每个切片是首切片、中间切片和尾切片均相同，
	// 仅需要图中的四个角部短点、四个边界节点和内部节点的不同，给出9种不同位置的差分方程。
	// 初始化
	// 1. 初始化网格划分的各个节点的初始温度
	// 在 calculator 中实现
	// todo 根据 参数中的 钢种从 jmatpro 接口获取对应的物性参数
	fmt.Println("钢种编号:", steelValue)

	// 2. 导热系数，200℃ 到 1600℃，随温度的上升而下降
	var LambdaStart = float32(500.0)
	var LambdaIter = float32(50.0-45.0) / ArrayLength
	for i := 0; i < ArrayLength; i++ {
		parameter.Lambda[i] = LambdaStart - float32(i)*LambdaIter
	}

	// 3. 密度
	var DensityStart = float32(8.0)
	var DensityIter = float32(8.0-7.0) / ArrayLength
	for i := 0; i < ArrayLength; i++ {
		parameter.Density[i] = DensityStart - float32(i)*DensityIter
	}

	// 4. 焓值
	var EnthalpyStart = float32(1000.0)
	var EnthalpyStep = float32(100) / ArrayLength
	for i := 0; i < ArrayLength; i++ {
		parameter.Enthalpy[i] = EnthalpyStart + float32(i)*EnthalpyStep
	}

	// 5. 综合换热系数
	var HEffStart = float32(5.0)
	var HEffStep = float32(20.0-15.0) / ArrayLength
	for i := 0; i < ArrayLength; i++ {
		parameter.HEff[i] = HEffStart + float32(i)*HEffStep
	}

	// 6. 热流密度
	var QStart = float32(4000.0)
	var QStep = float32(4000.0) / ArrayLength
	for i := 0; i < ArrayLength; i++ {
		parameter.Q[i] = QStart + float32(i)*QStep
	}

	// 7. 比热容
	var CStart = float32(10.0)
	var CStep = float32(1.0) / ArrayLength
	for i := 0; i < ArrayLength; i++ {
		parameter.C[i] = CStart + float32(i)*CStep
	}
}
