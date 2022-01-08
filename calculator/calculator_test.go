package calculator

import (
	"testing"
)

func TestCalculator(t *testing.T) {
	calculator := NewCalculator(0)
	calculator.Calculate()
}