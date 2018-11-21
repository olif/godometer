package main

import (
	"fmt"
	"os"
)

func main() {
	t := NewMonitoredTransfer(os.Stdin, os.Stdout)
	p := NewProgress(t.totalSize)

	updateTTY := func(val int64) {
		p.Update(val)
		fmt.Fprintf(os.Stderr, "%s", p.String())
	}
	t.AddListener(updateTTY)

	t.Start()
}
