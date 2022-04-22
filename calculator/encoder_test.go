package calculator

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestGeneratePushData(t *testing.T) {
	e := newEncoder()
	start := time.Now()
	res1 := e.GeneratePushData1()
	fmt.Println(time.Since(start), "dasdasdasd")
	res2 := e.GeneratePushData2()
	b1, _ := json.Marshal(&res1)
	b2, _ := json.Marshal(&res2)
	fmt.Println(len(b1), len(b2))
}
