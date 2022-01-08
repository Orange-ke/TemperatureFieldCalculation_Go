package server

import (
	"lz/calculator"
	"testing"
)

func TestBuildData(t *testing.T) {
	_ = calculator.NewCalculator(0)
	h := NewHub()
	h.c.BuildData()
}