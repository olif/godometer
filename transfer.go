package main

import (
	"io"
	"sync"
	"time"
)

// MonitoredTransfer transfers data between os.File while measuring progress
type MonitoredTransfer struct {
	reader    io.Reader
	writer    *countingWriter
	totalSize int64
	observers []func(TransferStats)
	startTime time.Time
}

// TransferStats represents the current state of the transfer
type TransferStats struct {
	transferredBytes int64
	elapsedTime      time.Duration
}

// NewMonitoredTransfer Returns a new instance of MonitoredTransfer
func NewMonitoredTransfer(in io.Reader, out io.Writer, totalSize int64) *MonitoredTransfer {
	w := &countingWriter{w: out}
	return &MonitoredTransfer{
		reader:    in,
		writer:    w,
		totalSize: totalSize,
		observers: []func(TransferStats){},
	}
}

// AddObserver adds a new observer. Each observer will be called in sync
func (t *MonitoredTransfer) AddObserver(cb func(TransferStats)) {
	t.observers = append(t.observers, cb)
}

// Start starts the transfer
func (t *MonitoredTransfer) Start() {
	ticker := time.NewTicker(100 * time.Millisecond)
	go func() {
		for range ticker.C {
			t.notify()
		}
	}()

	t.startTime = time.Now()
	io.Copy(t.writer, t.reader)
	ticker.Stop()
	t.notify()
}

func (t *MonitoredTransfer) notify() {
	stat := TransferStats{
		transferredBytes: t.writer.n(),
		elapsedTime:      time.Since(t.startTime),
	}
	for _, observer := range t.observers {
		observer(stat)
	}
}

type countingWriter struct {
	w            io.Writer
	writtenBytes int64
	err          error
	lock         sync.RWMutex
}

func (w *countingWriter) n() int64 {
	w.lock.RLock()
	defer w.lock.RUnlock()
	return w.writtenBytes
}

func (w *countingWriter) Write(p []byte) (n int, err error) {
	n, err = w.w.Write(p)
	w.lock.Lock()
	defer w.lock.Unlock()
	w.writtenBytes += int64(n)
	w.err = err
	return
}
