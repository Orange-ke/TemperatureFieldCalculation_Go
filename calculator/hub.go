package calculator

import (
	"fmt"
	"time"
)

type CalcHub struct {
	// 温度场推送
	Stop             chan struct{}
	PeriodCalcResult chan struct{}
	// 切片横截面温度数据推送
	PushSliceDetailRunning         bool
	StopPushSliceDataSignalForRun  chan struct{}
	StopPushSliceDataSignalForPush chan struct{}
	StopSuccessForRun              chan struct{}
	StopSuccessForPush             chan struct{}
	PeriodPushSliceData            chan struct{}
	// 切片纵截面
}

func NewCalcHub() *CalcHub {
	return &CalcHub{
		PeriodCalcResult: make(chan struct{}),

		PeriodPushSliceData:            make(chan struct{}),
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
	<-ch.StopSuccessForRun
	fmt.Println("stop running slice detail success")
	ch.StopPushSliceDataSignalForPush <- struct{}{}
	<-ch.StopSuccessForPush
	fmt.Println("stop push slice detail success")
}

// 横切面周期性推送任务
func (c *CalcHub) SliceDetailRun() {
LOOP:
	for {
		select {
		case <-c.StopPushSliceDataSignalForRun:
			fmt.Println("stop slice detail running")
			c.StopSuccessForRun <- struct{}{}
			break LOOP
		default:
			c.PushSliceDetailSignal()
			time.Sleep(1 * time.Second)
		}
	}
}

// 纵切面周期性推送任务
