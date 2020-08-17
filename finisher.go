package main

import (
	"sync"
)

type (
	Finisher struct {
		wg      *sync.WaitGroup
		input   <-chan *Build
		handler BuildHandler
	}
)

func (f *Finisher) Handle() {
	defer f.wg.Done()
	for b := range f.input {
		f.handler(b)
	}
}
func (f *Finisher) Wait() {
	f.wg.Wait()
}
