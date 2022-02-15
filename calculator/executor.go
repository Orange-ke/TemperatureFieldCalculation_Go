package calculator

import (
	"time"
)

type executor struct {
	dispatchChan chan task
	workers      int

	doneSoFar chan struct{}
	finish    chan struct{}
	start     chan task

	f func(t task)
}

type task struct {
	start int
	end   int
	deltaT float32
}

func newExecutor(workers int, f func(t task)) *executor {
	return &executor{
		dispatchChan: make(chan task, 50),
		workers:      workers,

		doneSoFar: make(chan struct{}, 50),
		finish:    make(chan struct{}, 1),
		start:     make(chan task, 1),

		f: f,
	}
}

func (e *executor) run() {
	total := 0
	totalTasks := 0
	doneSoFar := 0
	go func() {
		for {
			select {
			case tasks := <-e.start:
				//fmt.Println("master 分配任务: ", tasks)
				if tasks.end - tasks.start == 0 {
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
						totalTasks = e.workers*2
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
						for start < total - remainder {
							e.dispatchChan <- task{start: start, end: start + 1, deltaT: tasks.deltaT}
							start++
						}
					} else {
						half1, half2 := taskLen/2, taskLen/2
						if taskLen%2 == 1 {
							half2++
						}
						for start < total - remainder {
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
					e.f(t)
					e.doneSoFar <- struct{}{}
					//fmt.Println("worker ", i, "完成任务: ", t)
				default:
					time.Sleep(1 * time.Millisecond)
				}
			}
		}(i)
	}
}
