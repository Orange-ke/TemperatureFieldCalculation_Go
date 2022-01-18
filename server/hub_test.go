package server

import (
	"lz/calculator"
	"testing"
)

func TestBuildData(t *testing.T) {
	c := calculator.NewCalcHub()
	h := NewHub()
	h.c = c
	h.c.BuildData()
}