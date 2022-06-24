package calculator

import (
	"fmt"
	"testing"
)

func TestNewSteel(t *testing.T) {
	steel := NewSteel(1, &CastingMachine{})
	fmt.Println(steel.Parameter.Enthalpy2Temp(1.3246079e+06))
	fmt.Println(steel.Parameter.Temp2Enthalpy(1599.9827))
	fmt.Println(steel.Parameter.Emissivity[1153])

	fmt.Println(calculateHbr(1153, 70, steel.Parameter))
}