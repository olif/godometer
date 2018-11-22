package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	var totalSize int64
	fIn, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	if fIn.Mode().IsRegular() {
		totalSize = fIn.Size()
	}

	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)

	t := NewMonitoredTransfer(reader, writer, totalSize)
	p := NewProgress(t.totalSize)

	updateTTY := func(stats TransferStats) {
		p.Update(stats)
		fmt.Fprintf(os.Stderr, "%s", p.String())
	}
	t.AddObserver(updateTTY)

	t.Start()
}
