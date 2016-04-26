package worker

import (
	"sync/atomic"
	"time"
)

// Worker represents the interface that mechanisms performing periodic background tasks should conform to
type Worker interface {
	Start()
	Stop(wait time.Duration)
	Started() bool
}

type repeatWorker struct {
	started      uint32
	stopCh       chan chan struct{}
	repeatTicker *time.Ticker
	do           func()
}

// NewRepeat returns a new instance of a worker that executes the desired function repeatedly
func NewRepeat(repeat time.Duration, do func()) Worker {
	return &repeatWorker{
		do:           do,
		stopCh:       make(chan chan struct{}, 1),
		repeatTicker: time.NewTicker(repeat),
	}
}

func (w *repeatWorker) Start() {
	if atomic.SwapUint32(&w.started, 1) == 1 {
		return
	}
	go func() {
		defer atomic.StoreUint32(&w.started, 0)
		for {
			select {
			case ch := <-w.stopCh:
				ch <- struct{}{}
				return
			default:
			}
			go w.do()
			<-w.repeatTicker.C
		}
	}()
}

func (w *repeatWorker) Stop(wait time.Duration) {
	if w.Started() {
		ch := make(chan struct{})
		w.stopCh <- ch
		select {
		case <-ch:
		case <-time.After(wait):
		}
	}
}

func (w *repeatWorker) Started() bool {
	return atomic.LoadUint32(&w.started) != 0
}
