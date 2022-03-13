package calculator

import (
	"time"
)

type executor interface {
	run(c *calculatorWithArrDeque)
	dispatchTask(deltaT float32, first, last int) time.Duration
}

// 基于切片任务分配
type executorBaseOnSlice struct {
	dispatchChan chan task
	workers      int

	doneSoFar chan struct{}
	finish    chan struct{}
	start     chan task
}

type task struct {
	start  int
	end    int
	deltaT float32
}

func newExecutorBaseOnSlice(workers int) *executorBaseOnSlice {
	e := &executorBaseOnSlice{
		dispatchChan: make(chan task, 50),
		workers:      workers,

		doneSoFar: make(chan struct{}, 50),
		finish:    make(chan struct{}, 1),
		start:     make(chan task, 1),
	}

	return e
}

func (e *executorBaseOnSlice) dispatchTask(deltaT float32, first, last int) time.Duration {
	//fmt.Println("calculate start")
	start := time.Now()
	e.start <- task{start: first, end: last, deltaT: deltaT}
	//fmt.Println("task dispatched")
	<-e.finish
	//fmt.Println("task finished")
	return time.Since(start)
}

func (e *executorBaseOnSlice) run(c *calculatorWithArrDeque) {
	total := 0
	totalTasks := 0
	doneSoFar := 0
	go func() {
		for {
			select {
			case tasks := <-e.start:
				//fmt.Println("master 分配任务: ", tasks)
				if tasks.end-tasks.start == 0 {
					e.finish <- struct{}{}
					break
				}
				total = tasks.end - tasks.start
				taskLen, remainder := total/e.workers, total%e.workers
				if taskLen == 0 {
					totalTasks = remainder
				} else if remainder == 0 {
					if taskLen == 1 {
						totalTasks = e.workers
					} else {
						totalTasks = e.workers * 2
					}
				} else {
					if taskLen == 1 {
						totalTasks = e.workers + remainder
					} else {
						totalTasks = e.workers*2 + remainder
					}
				}

				start := 0
				if taskLen > 0 {
					if taskLen == 1 {
						for start < total-remainder {
							e.dispatchChan <- task{start: start, end: start + 1, deltaT: tasks.deltaT}
							start++
						}
					} else {
						half1, half2 := taskLen/2, taskLen/2
						if taskLen%2 == 1 {
							half2++
						}
						for start < total-remainder {
							if half1 != 0 {
								e.dispatchChan <- task{start: start, end: start + half1, deltaT: tasks.deltaT}
								start += half1
							}
							if half2 != 0 {
								e.dispatchChan <- task{start: start, end: start + half2, deltaT: tasks.deltaT}
								start += half2
							}
						}
					}
				}

				for i := 0; i < remainder; i++ {
					e.dispatchChan <- task{start: start, end: start + 1, deltaT: tasks.deltaT}
					start++
				}
			case <-e.doneSoFar:
				doneSoFar++
				if doneSoFar == totalTasks {
					e.finish <- struct{}{}
					doneSoFar = 0
				}
			default:
				time.Sleep(1 * time.Millisecond)
			}
		}
	}()

	for i := 0; i < e.workers; i++ {
		go func(i int) {
			for {
				select {
				case t := <-e.dispatchChan:
					//fmt.Println("worker ", i, "获取到任务: ", t)
					e.traverseSpirally(t, c)
					e.doneSoFar <- struct{}{}
					//fmt.Println("worker ", i, "完成任务: ", t)
				default:
					time.Sleep(1 * time.Millisecond)
				}
			}
		}(i)
	}
}

// 直接调度，直接进行遍历
type executorBaseOnBlock struct {
	step         int
	edgeWidth    int
	dispatchChan chan task
	finishChan   chan struct{}
	f            []func(t task, c *calculatorWithArrDeque)
}

func newExecutorBaseOnBlock(edgeWidth int) *executorBaseOnBlock {
	if edgeWidth < 0 {
		edgeWidth = 0
	}
	if edgeWidth > 20 {
		edgeWidth = 20
	}

	e := &executorBaseOnBlock{
		edgeWidth:    edgeWidth,
		dispatchChan: make(chan task, 1),
		finishChan:   make(chan struct{}, 10),
		f: make([]func(t task, c *calculatorWithArrDeque), 4),
	}

	e.step = 1
	if e.edgeWidth > 0 {
		e.step = 2
	}

	e.f[0] = e.calculateCase1
	e.f[1] = e.calculateCase2
	e.f[2] = e.calculateCase3
	e.f[3] = e.calculateCase4

	return e
}

func (e *executorBaseOnBlock) run(c *calculatorWithArrDeque) {
	for i := 0; i < 4; i++ {
		go func(i int) {
			for {
				select {
				case t := <-e.dispatchChan:
					e.f[i](t, c)
					e.finishChan <- struct{}{}
				default:
					time.Sleep(time.Millisecond)
				}
			}
		}(i)
	}
}

func (e *executorBaseOnBlock) dispatchTask(deltaT float32, first, last int) time.Duration {
	start := time.Now()
	t := task{
		start:  first,
		end:    last,
		deltaT: deltaT,
	}
	for i := 0; i < 4; i++ {
		e.dispatchChan <- t
	}

	for i := 0; i < 4; i++ {
		<-e.finishChan
	}
	return time.Since(start)
}
