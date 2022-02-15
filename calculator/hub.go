package calculator

import "fmt"

type CalcHub struct {
	Stop             chan struct{}
	PeriodCalcResult chan struct{}

	PushSliceDetailRunning         bool
	StopPushSliceDataSignalForRun  chan struct{}
	StopPushSliceDataSignalForPush chan struct{}
	StopSuccessForRun              chan struct{}
	StopSuccessForPush             chan struct{}
	PeriodPushSliceData            chan struct{}
}

func NewCalcHub() *CalcHub {
	return &CalcHub{
		PeriodCalcResult:    make(chan struct{}),
		PeriodPushSliceData: make(chan struct{}),

		StopPushSliceDataSignalForRun:  make(chan struct{}, 10),
		StopPushSliceDataSignalForPush: make(chan struct{}, 10),
		StopSuccessForRun:              make(chan struct{}, 10),
		StopSuccessForPush:             make(chan struct{}, 10),
	}
}

// 温度场计算
func (ch *CalcHub) PushSignal() {
	ch.PeriodCalcResult <- struct{}{}
}

func (ch *CalcHub) StopSignal() {
	close(ch.Stop)
}

func (ch *CalcHub) StartSignal() {
	ch.Stop = make(chan struct{})
}

// 切片详情数据
func (ch *CalcHub) PushSliceDetailSignal() {
	ch.PeriodPushSliceData <- struct{}{}
}

func (ch *CalcHub) StopPushSliceDetail() {
	fmt.Println("start to stop push slice detail")
	ch.StopPushSliceDataSignalForRun <- struct{}{}
	<- ch.StopSuccessForRun
	ch.StopPushSliceDataSignalForPush <- struct{}{}
	<- ch.StopSuccessForPush
	fmt.Println("stop push slice detail success")
}
