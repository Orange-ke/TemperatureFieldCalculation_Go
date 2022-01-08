package calculator

type CalcHub struct {
	Stop chan struct{}
	PeriodCalcResult chan struct{}

}

func NewCalcHub() *CalcHub {
	return &CalcHub{
		PeriodCalcResult: make(chan struct{}),
		Stop: make(chan struct{}),
	}
}

func (ch *CalcHub) PushSignal() {
	ch.PeriodCalcResult <- struct{}{}
}

func (ch *CalcHub) StopSignal() {
	close(ch.Stop)
}
