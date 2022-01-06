package calculator

import "testing"

func TestCalculator(t *testing.T) {
	calculator := NewCalculator(30)
	calculator.Calculate()
}