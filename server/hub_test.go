package server

import (
	"lz/calculator"
	"testing"
)

func TestBuildData(t *testing.T) {
	c := calculator.NewCalculator(0)
	h := NewHub()
	h.c = c
	h.c.BuildData()
}