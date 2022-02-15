package calculator

import (
	"testing"
)

func TestCalculator(t *testing.T) {
	calculator := NewCalculatorWithArrDeque(0)
	calculator.Calculate()
}

func TestCalculateTimeStep(t *testing.T) {
	calculator := NewCalculatorWithArrDeque(0)
	calculator.calculateTimeStep()
}

//func TestCalculator_Test(t *testing.T) {
//	calculator := NewCalculatorWithArrDeque(0)
//	calculator.Calculate()
//}
