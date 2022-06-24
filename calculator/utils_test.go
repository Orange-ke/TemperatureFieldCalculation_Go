package calculator

import (
	"fmt"
	"testing"
)

// 测试水的综合换热系数
func TestHOfWater(t *testing.T) {
	fmt.Println(1/ROfWater(3000.0, 0.005, 36.5))
}

func TestHOfAll(t *testing.T) {
	fmt.Println(1 / (ROfWater(3000.0, 0.005, 36.5) + ROfCu() + 1/2000.0))
}

func TestCalculateVwt(t *testing.T) {
	fmt.Println(CalculateVwt(3000.0, 0.005))
}

// Hbr
func TestCalculateHbr(t *testing.T) {
	fmt.Println(calculateHbr_(1200, 70) * (1153 - 37))
}

// 辐射换热系数产生的热流密度
func TestCalculateQar(t *testing.T) {
	fmt.Println(calculateQar(800, 70))
}

// 计算固相率
func TestCalculateSolidFraction(t *testing.T) {
	fmt.Println(calculateSolidFraction(1430.1, 1429.76,1499.1))
}
