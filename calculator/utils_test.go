package calculator

import (
	"fmt"
	"testing"
)

// 测试水的综合换热系数
func TestHOfWater(t *testing.T) {
	fmt.Println(1/ROfWater())
	fmt.Println(1 / (ROfWater() + ROfCu() + 1/2000.0))
}

// 测试直接喷淋区域的综合换热系数计算

// 测试AB，CD区域综合换热系数计算

// 测试辊子直接接触区域换热系数计算

// 测试空气换热系数计算

// 测试加权平均值
