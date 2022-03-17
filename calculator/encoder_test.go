package calculator

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestGeneratePushData(t *testing.T) {
	e := newEncoder()
	res1 := e.GeneratePushData1()
	res2 := e.GeneratePushData2()
	b1, _ := json.Marshal(&res1)
	b2, _ := json.Marshal(&res2)
	fmt.Println(len(b1), len(b2))
}
