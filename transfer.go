package main

import (
	"bufio"
	"io"
	"os"
	"time"
)

// ObservableLong is an interface that allows a measurement of type long to be read
type ObservableLong interface {
	N() int64
}

// Sampler samples Observables
type Sampler struct {
	stopped   bool
	endSignal chan bool
}

// NewSampler returns a new sampler
func NewSampler() *Sampler {
	return &Sampler{
		stopped:   false,
		endSignal: make(chan bool),
	}
}

func (s *Sampler) Stop() {
	s.endSignal <- true
}

// Sample starts sampling of the observable
func (s *Sampler) Sample(periodMs time.Duration, o ObservableLong) <-chan int64 {
	c := make(chan int64, 100)
	go func() {
		for !s.stopped {
			select {
			case <-time.After(periodMs * time.Millisecond):
				c <- o.N()
			case <-s.endSignal:
				s.stopped = true
				close(c)
			}
		}
	}()

	return c
}

type MonitoredTransfer struct {
	reader    io.Reader
	writer    *CountingWriter
	sampler   *Sampler
	totalSize int64
	listeners []ValueCallback
}

func NewMonitoredTransfer(in *os.File, out *os.File) *MonitoredTransfer {
	var totalSize int64
	fIn, err := in.Stat()
	if err != nil {
		panic(err)
	}

	if fIn.Mode().IsRegular() {
		totalSize = fIn.Size()
	}

	r := bufio.NewReader(in)
	w := NewCountingWriter(out)
	return &MonitoredTransfer{
		reader:    r,
		writer:    w,
		totalSize: totalSize,
		listeners: []ValueCallback{},
	}
}

type ValueCallback func(int64)

func (t *MonitoredTransfer) AddListener(cb ValueCallback) {
	t.listeners = append(t.listeners, cb)
}

func (t *MonitoredTransfer) Start() {
	sampler := NewSampler()
	valChan := sampler.Sample(100, t.writer)
	go func() {
		for val := range valChan {
			t.notify(val)
		}
	}()
	io.Copy(t.writer, t.reader)
	sampler.Stop()
	val := t.writer.N()
	t.notify(val)
}

func (t *MonitoredTransfer) notify(val int64) {
	for _, listener := range t.listeners {
		listener(val)
	}
}
