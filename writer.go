package main

import (
	"io"
	"sync"
)

// CountingWriter writer that stores the number of written bytes
type CountingWriter struct {
	w            io.Writer
	writtenBytes int64
	err          error
	lock         sync.RWMutex
}

// N returns the number of bytes written
func (w *CountingWriter) N() int64 {
	w.lock.RLock()
	defer w.lock.RUnlock()
	return w.writtenBytes
}

// NewCountingWriter returns a counting writer
func NewCountingWriter(w io.Writer) *CountingWriter {
	return &CountingWriter{
		w: w,
	}
}

func (w *CountingWriter) Write(p []byte) (n int, err error) {
	n, err = w.w.Write(p)
	w.lock.Lock()
	defer w.lock.Unlock()
	w.writtenBytes += int64(n)
	w.err = err
	return
}
