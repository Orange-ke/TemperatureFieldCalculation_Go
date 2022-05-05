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
		left, right, top, bottom := 0, Length/XStep-1, 0, Width/YStep-1 // 每个切片迭代时需要重置
		// parameter set
		parameter = c.getParameter(z)
		// 计算最外层， 逆时针
		{
			// 1. 三个顶点，左下方顶点仅当其外一层温度不是初始温度时才开始计算
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
			// 逆时针螺旋遍历
			for left <= right && top <= bottom {
				if item[0][right] != item[0][right+1] ||
					item[0][right] != item[0][right-1] ||
					item[0][right] != item[1][right] {
					c.calculatePointBA(t.deltaT, right, z, item, parameter)
					count++
				}
				for row := top + 1; row <= bottom; row++ {
					// [row][right]
					if item[row][right] != item[row][right+1] ||
						item[row][right] != item[row][right-1] ||
						item[row][right] != item[row+1][right] ||
						item[row][right] != item[row-1][right] {
						c.calculatePointIN(t.deltaT, right, row, z, item, parameter)
						count++
					}
				}
				if left < right && top < bottom {
					for column := right - 1; column > left; column-- {
						// [bottom][column]
						if item[bottom][column] != item[bottom][column+1] ||
							item[bottom][column] != item[bottom][column-1] ||
							item[bottom][column] != item[bottom+1][column] ||
							item[bottom][column] != item[bottom-1][column] {
							c.calculatePointIN(t.deltaT, column, bottom, z, item, parameter)
							count++
						}
						if item[bottom][0] == item[bottom+1][0] ||
							item[bottom][0] == item[bottom-1][0] ||
							item[bottom][0] == item[bottom][1] {
							c.calculatePointLA(t.deltaT, bottom, z, item, parameter)
							count++
						}
					}
				}
				if top == bottom {
					if item[0][0] != item[0][1] || item[0][0] != item[1][0] {
						c.calculatePointLB(t.deltaT, z, item, parameter)
						count++
					}
					for column := right - 1; column > left; column-- {
						if item[0][column] != item[0][column+1] ||
							item[0][column] != item[0][column-1] ||
							item[0][column] != item[1][column] {
							c.calculatePointBA(t.deltaT, column, z, item, parameter)
							count++
						}
					}
				}
				right--
				bottom--
			}
		}
		//fmt.Println("消耗时间: ", time.Since(start), "计算的点数: ", count, "实际需要遍历的点数: ", (t.end-t.start)*11340)
	})
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
		for i := 1; i < Length/XStep/2; i++ {
			c.calculatePointTA(t.deltaT, i, z, item, parameter)
			count++
		}
		for j := Width / YStep / 2; j < Width/YStep-1; j++ {
			c.calculatePointLA(t.deltaT, j, z, item, parameter)
			count++
		}
		for j := Width/YStep - 1 - e.edgeWidth; j < Width/YStep-1; j++ {
			for i := 1; i < 1+e.edgeWidth; i++ {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := Width / YStep / 2; j < Width/YStep-1-e.edgeWidth; j++ {
			for i := 1; i < 1+e.edgeWidth; i++ {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := Width/YStep - 1 - e.edgeWidth; j < Width/YStep-1; j++ {
			for i := 1 + e.edgeWidth; i < Length/XStep/2; i = i + 1 {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := Width / YStep / 2; j < Width/YStep-1-e.edgeWidth; j = j + e.step {
			for i := 1 + e.edgeWidth; i < Length/XStep/2; i = i + e.step {
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
		for i := Length / XStep / 2; i < Length/XStep-1; i++ {
			c.calculatePointTA(t.deltaT, i, z, item, parameter)
			count++
		}
		for j := Width / YStep / 2; j < Width/YStep-1; j++ {
			c.calculatePointRA(t.deltaT, j, z, item, parameter)
			count++
		}
		for j := Width/YStep - 1 - e.edgeWidth; j < Width/YStep-1; j++ {
			for i := Length/XStep - 1 - e.edgeWidth; i < Length/XStep-1; i++ {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := Width / YStep / 2; j < Width/YStep-1-e.edgeWidth; j++ {
			for i := Length/XStep - 1 - e.edgeWidth; i < Length/XStep-1; i++ {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := Width/YStep - 1 - e.edgeWidth; j < Width/YStep-1; j++ {
			for i := Length / XStep / 2; i < Length/XStep-1-e.edgeWidth; i = i + 1 {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := Width / YStep / 2; j < Width/YStep-1-e.edgeWidth; j = j + e.step {
			for i := Length / XStep / 2; i < Length/XStep-1-e.edgeWidth; i = i + e.step {
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
		for i := Length / XStep / 2; i < Length/XStep-1; i++ {
			c.calculatePointBA(t.deltaT, i, z, item, parameter)
			count++
		}
		for j := 1; j < Width/YStep/2; j++ {
			c.calculatePointRA(t.deltaT, j, z, item, parameter)
			count++
		}
		for j := 1; j < 1+e.edgeWidth; j++ {
			for i := Length/XStep - 1 - e.edgeWidth; i < Length/XStep-1; i++ {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := 1 + e.edgeWidth; j < Width/YStep/2; j++ {
			for i := Length/XStep - 1 - e.edgeWidth; i < Length/XStep-1; i++ {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := 1; j < 1+e.edgeWidth; j++ {
			for i := Length / XStep / 2; i < Length/XStep-1-e.edgeWidth; i++ {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := 1 + e.edgeWidth; j < Width/YStep/2; j = j + e.step {
			for i := Length / XStep / 2; i < Length/XStep-1-e.edgeWidth; i = i + e.step {
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
		for i := 1; i < Length/XStep/2; i++ {
			c.calculatePointBA(t.deltaT, i, z, item, parameter)
			count++
		}
		for j := 1; j < Width/YStep/2; j++ {
			c.calculatePointLA(t.deltaT, j, z, item, parameter)
			count++
		}
		for j := 1; j < 1+e.edgeWidth; j++ {
			for i := 1; i < 1+e.edgeWidth; i++ {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := 1 + e.edgeWidth; j < Width/YStep/2; j++ {
			for i := 1; i < 1+e.edgeWidth; i++ {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := 1; j < 1+e.edgeWidth; j++ {
			for i := 1 + e.edgeWidth; i < Length/XStep/2; i++ {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
		for j := 1 + e.edgeWidth; j < Width/YStep/2; j = j + e.step {
			for i := 1 + e.edgeWidth; i < Length/XStep/2; i = i + e.step {
				c.calculatePointIN(t.deltaT, i, j, z, item, parameter)
				count++
			}
		}
	})
	fmt.Println("任务4执行时间: ", time.Since(start), "总共计算：", count, "个点")
}
