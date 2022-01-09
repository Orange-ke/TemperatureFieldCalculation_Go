package calculator

type CalcHub struct {
	Stop             chan struct{}
	PeriodCalcResult chan struct{}
}

func NewCalcHub() *CalcHub {
	return &CalcHub{
		PeriodCalcResult: make(chan struct{}),
	}
}

func (ch *CalcHub) PushSignal() {
	ch.PeriodCalcResult <- struct{}{}
}

func (ch *CalcHub) StopSignal() {
	close(ch.Stop)
}

func (ch *CalcHub) StartSignal() {
	ch.Stop = make(chan struct{})
}
