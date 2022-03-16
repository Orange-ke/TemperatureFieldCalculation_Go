package calculator

import (
	"fmt"
	"lz/model"
	"time"
)

func (e *executorBaseOnSlice) traverseSpirally(t task, c *calculatorWithArrDeque) {
	//start := time.Now()
	count := 0
	var parameter *Parameter
	c.Field.TraverseSpirally(t.start, t.end, func(z int, item *model.ItemType) {
		// 跳过为空的切片， 即值为-1
		if item[0][0] == -1 {
			return
		}
		left, right, top, bottom := 0, model.Length/model.XStep-1, 0, model.Width/model.YStep-1 // 每个切片迭代时需要重置
		// parameter set
		parameter = c.getParameter(z)
		// 计算最外层， 逆时针
		// 1. 三个顶点，左下方顶点仅当其外一层温度不是初始温度时才开始计算
		//c.calculatePointLB(t.deltaT, z, item) 不用计算，因为还未产生温差
		{
			c.calculatePointRB(t.deltaT, z, item, parameter)
			c.calculatePointRT(t.deltaT, z, item, parameter)
			c.calculatePointLT(t.deltaT, z, item, parameter)
			count += 3
			for row := top + 1; row < bottom; row++ {
				// [row][right]
				c.calculatePointRA(t.deltaT, row, z, item, parameter)
				count++
			}
			for column := right - 1; column > left; column-- {
				// [bottom][column]
				c.calculatePointTA(t.deltaT, column, z, item, parameter)
				count++
			}
			right--
			bottom--
		}

		{
			//stop := 0 // 当出现两层的温度都是 初始温度时，则停止遍历
			//allNotChanged := true
			// 逆时针
			for left <= right && top <= bottom {
				c.calculatePointBA(t.deltaT, right, z, item, parameter)
				//if item[0][right] != c.castingMachine.CoolerConfig.StartTemperature {
				//	allNotChanged = false
				//}
				count++
				for row := top + 1; row <= bottom; row++ {
					// [row][right]
					c.calculatePointIN(t.deltaT, right, row, z, item, parameter)
					//if item[row][right] != c.castingMachine.CoolerConfig.StartTemperature {
					//	allNotChanged = false
					//}
					count++
				}
				if left < right && top < bottom {
					for column := right - 1; column > left; column-- {
						// [bottom][column]
						c.calculatePointIN(t.deltaT, column, bottom, z, item, parameter)
						//if item[bottom][column] == c.castingMachine.CoolerConfig.StartTemperature {
						//	allNotChanged = false
						//}
						count++
					}
					c.calculatePointLA(t.deltaT, bottom, z, item, parameter)
					//if item[bottom][0] != c.castingMachine.CoolerConfig.StartTemperature {
					//	allNotChanged = false
					//}
					count++
				}
				if top == bottom {
					c.calculatePointLB(t.deltaT, z, item, parameter)
					count++
					for column := right - 1; column > left; column-- {
						c.calculatePointBA(t.deltaT, column, z, item, parameter)
						count++
					}
				}
				right--
				bottom--
				// 如果该层与其的温度都未改变，即都是初始温度则计数
				//if allNotChanged {
				//	stop++
				//	allNotChanged = true
				//	if stop == 2 {
				//		break
				//	}
				//}
			}
		}
	})
	//fmt.Println("消耗时间: ", time.Since(start), "计算的点数: ", count, "实际需要遍历的点数: ", (t.end-t.start)*11340)
}

// 分块遍历
func (e *executorBaseOnBlock) calculateCase1(t task, c *calculatorWithArrDeque) {
	var start = time.Now()
	var count = 0
	var parameter *Parameter
	fmt.Println("开始计算case1")
	c.Field.Traverse(func(z int, item *model.ItemType) {
		// parameter set
		parameter = c.getParameter(z)
		// 先计算点，再计算外表面，再计算里面的点
		c.calculatePointLT(t.deltaT, z, item, parameter)
		count++
		for i := 1; i < model.Length/model.XStep/2; i++ {
			c.calculatePointTA(t.deltaT, i, z, item, parameter)
			count++
		}
		for j := model.Width / model.YStep / 2; j < model.Width/model.YStep-1; j++ {
			c.calculatePointLA(t.deltaT, j, z, item, parameter)
			count++
		}
		for j := model.Width/model.YStep - 1 - e.edgeWidth; j < model.Width/model.YStep-1; j++ {
			for i := 1; i < 1+e.edgeWidth; i++ {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := model.Width / model.YStep / 2; j < model.Width/model.YStep-1-e.edgeWidth; j++ {
			for i := 1; i < 1+e.edgeWidth; i++ {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := model.Width/model.YStep - 1 - e.edgeWidth; j < model.Width/model.YStep-1; j++ {
			for i := 1 + e.edgeWidth; i < model.Length/model.XStep/2; i = i + 1 {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := model.Width / model.YStep / 2; j < model.Width/model.YStep-1-e.edgeWidth; j = j + e.step {
			for i := 1 + e.edgeWidth; i < model.Length/model.XStep/2; i = i + e.step {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
	})

	fmt.Println("任务1执行时间: ", time.Since(start), "总共计算：", count, "个点")
}

func (e *executorBaseOnBlock) calculateCase2(t task, c *calculatorWithArrDeque) {
	var start = time.Now()
	var count = 0
	var parameter *Parameter
	fmt.Println("开始计算case2")
	c.Field.Traverse(func(z int, item *model.ItemType) {
		// parameter set
		parameter = c.getParameter(z)
		// 先计算点，再计算外表面，再计算里面的点
		c.calculatePointRT(t.deltaT, z, item, parameter)
		count++
		for i := model.Length / model.XStep / 2; i < model.Length/model.XStep-1; i++ {
			c.calculatePointTA(t.deltaT, i, z, item, parameter)
			count++
		}
		for j := model.Width / model.YStep / 2; j < model.Width/model.YStep-1; j++ {
			c.calculatePointRA(t.deltaT, j, z, item, parameter)
			count++
		}
		for j := model.Width/model.YStep - 1 - e.edgeWidth; j < model.Width/model.YStep-1; j++ {
			for i := model.Length/model.XStep - 1 - e.edgeWidth; i < model.Length/model.XStep-1; i++ {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := model.Width / model.YStep / 2; j < model.Width/model.YStep-1-e.edgeWidth; j++ {
			for i := model.Length/model.XStep - 1 - e.edgeWidth; i < model.Length/model.XStep-1; i++ {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := model.Width/model.YStep - 1 - e.edgeWidth; j < model.Width/model.YStep-1; j++ {
			for i := model.Length / model.XStep / 2; i < model.Length/model.XStep-1-e.edgeWidth; i = i + 1 {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := model.Width / model.YStep / 2; j < model.Width/model.YStep-1-e.edgeWidth; j = j + e.step {
			for i := model.Length / model.XStep / 2; i < model.Length/model.XStep-1-e.edgeWidth; i = i + e.step {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
	})
	fmt.Println("任务2执行时间: ", time.Since(start), "总共计算：", count, "个点")
}

func (e *executorBaseOnBlock) calculateCase3(t task, c *calculatorWithArrDeque) {
	var start = time.Now()
	var count = 0
	var parameter *Parameter
	fmt.Println("开始计算case3")
	c.Field.Traverse(func(z int, item *model.ItemType) {
		// parameter set
		parameter = c.getParameter(z)
		// 先计算点，再计算外表面，再计算里面的点
		c.calculatePointRB(t.deltaT, z, item, parameter)
		count++
		for i := model.Length / model.XStep / 2; i < model.Length/model.XStep-1; i++ {
			c.calculatePointBA(t.deltaT, i, z, item, parameter)
			count++
		}
		for j := 1; j < model.Width/model.YStep/2; j++ {
			c.calculatePointRA(t.deltaT, j, z, item, parameter)
			count++
		}
		for j := 1; j < 1+e.edgeWidth; j++ {
			for i := model.Length/model.XStep - 1 - e.edgeWidth; i < model.Length/model.XStep-1; i++ {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := 1 + e.edgeWidth; j < model.Width/model.YStep/2; j++ {
			for i := model.Length/model.XStep - 1 - e.edgeWidth; i < model.Length/model.XStep-1; i++ {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := 1; j < 1+e.edgeWidth; j++ {
			for i := model.Length / model.XStep / 2; i < model.Length/model.XStep-1-e.edgeWidth; i++ {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := 1 + e.edgeWidth; j < model.Width/model.YStep/2; j = j + e.step {
			for i := model.Length / model.XStep / 2; i < model.Length/model.XStep-1-e.edgeWidth; i = i + e.step {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
	})
	fmt.Println("任务3执行时间: ", time.Since(start), "总共计算：", count, "个点")
}

func (e *executorBaseOnBlock) calculateCase4(t task, c *calculatorWithArrDeque) {
	var start = time.Now()
	var count = 0
	var parameter *Parameter
	fmt.Println("开始计算case4")
	c.Field.Traverse(func(z int, item *model.ItemType) {
		// parameter set
		parameter = c.getParameter(z)
		// 先计算点，再计算外表面，再计算里面的点
		c.calculatePointLB(t.deltaT, z, item, parameter)
		count++
		for i := 1; i < model.Length/model.XStep/2; i++ {
			c.calculatePointBA(t.deltaT, i, z, item, parameter)
			count++
		}
		for j := 1; j < model.Width/model.YStep/2; j++ {
			c.calculatePointLA(t.deltaT, j, z, item, parameter)
			count++
		}
		for j := 1; j < 1+e.edgeWidth; j++ {
			for i := 1; i < 1+e.edgeWidth; i++ {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := 1 + e.edgeWidth; j < model.Width/model.YStep/2; j++ {
			for i := 1; i < 1+e.edgeWidth; i++ {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := 1; j < 1+e.edgeWidth; j++ {
			for i := 1 + e.edgeWidth; i < model.Length/model.XStep/2; i++ {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := 1 + e.edgeWidth; j < model.Width/model.YStep/2; j = j + e.step {
			for i := 1 + e.edgeWidth; i < model.Length/model.XStep/2; i = i + e.step {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
	})
	fmt.Println("任务4执行时间: ", time.Since(start), "总共计算：", count, "个点")
}
