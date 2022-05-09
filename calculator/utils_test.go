package calculator

import (
	"fmt"
	"testing"
)

func TestHOfWater(t *testing.T) {
	fmt.Println(1/ROfWater())
	fmt.Println(1 / (ROfWater() + ROfCu() + 1/1200.0))
}