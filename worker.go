package main

import (
	"sync"
)

type (
	Worker struct {
		name    string
		wg      *sync.WaitGroup
		input   <-chan *Build
		output  chan<- *Build
		handler BuildHandler
	}
)

// pool is a wrapper that allows to process a chain in a pool. It consumes all
// builds from `input` calls `handler` on them, decremts their wg and puts the
// build in `ouput`
func (w *Worker) pool(size int) {
	p := make(chan bool, size)
	for i := 0; i < size; i++ {
		p <- true
	}

	defer w.wg.Done()
	for b := range w.input {
		w.wg.Add(1)
		lock := <-p
		go func(build *Build, lock bool) {
			defer w.wg.Done()
			w.handler(build)
			p <- lock
			w.output <- build
		}(b, lock)
	}
}

func (w *Worker) WaitAndClose() {
	w.wg.Wait()
	close(w.output)
}
