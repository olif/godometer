package main

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
	"time"

	pbar "github.com/tj/go-progress"
	spin "github.com/tj/go-spin"
	"golang.org/x/sys/unix"
)

// Progress is tty progress indicator
type Progress interface {
	Update(stats TransferStats)
	String() string
}

// FiniteProgress represents a progress indicator with a known ending state
type FiniteProgress struct {
	progressBar *pbar.Bar
	size        int64
	currentStat TransferStats
}

// Update updates progress with current value
func (p *FiniteProgress) Update(v TransferStats) {
	p.currentStat = v
	p.progressBar.ValueInt(int(v.transferredBytes))
}

// String returns a tty representation of current progres
func (p *FiniteProgress) String() string {
	a := fmt.Sprintf("%s %s", byteCountBinary(p.currentStat.transferredBytes), p.progressBar.String())
	return CenterLine(a)
}

// InfiniteProgress represents a progress indicator without a known ending state
type InfiniteProgress struct {
	currentStat TransferStats
	spinner     *spin.Spinner
}

// Update updates progress with current value
func (p *InfiniteProgress) Update(v TransferStats) {
	p.currentStat = v
}

// String returns a tty representation of current progres
func (p *InfiniteProgress) String() string {
	avgSpeed := getAverageSpeed(p.currentStat.transferredBytes, p.currentStat.elapsedTime)
	speedString := fmt.Sprintf("%s / s", byteCountBinary(int64(avgSpeed)))
	return fmt.Sprintf("\r  \033[36m\033[m %s transfering: %s, %s", p.spinner.Next(), byteCountBinary(p.currentStat.transferredBytes), speedString)
}

// NewProgress returns a finite progress indicator if totalSize > 0 otherwise, an infinite progress indicator
func NewProgress(totalSize int64) Progress {
	var progress Progress
	if totalSize > 0 {
		b := pbar.NewInt(int(totalSize))
		b.Width = getWidth() - 20
		progress = &FiniteProgress{
			progressBar: b,
			size:        totalSize,
			currentStat: TransferStats{},
		}
	} else {
		progress = &InfiniteProgress{
			spinner:     spin.New(),
			currentStat: TransferStats{},
		}
	}

	return progress
}

func getAverageSpeed(transferredBytes int64, elapsedTime time.Duration) float64 {
	if elapsedTime == 0 {
		return 0
	}
	return float64(transferredBytes) / (elapsedTime.Seconds())
}

func byteCountBinary(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}

	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func digitsInInt(val int64) int {
	return int(math.Floor(math.Log10(float64(val))))
}

func CenterLine(s string) string {
	r := strings.Repeat
	w := getWidth()
	s = "\n" + s
	//h := getHeight()
	xpad := int(math.Abs(float64((int(w) - Length(s)) / 2)))
	ypad := 1 // int(h / 2)
	MoveUp(3)
	return r("\n", ypad) + r(" ", xpad) + s + r("\n", ypad)

}

func MoveUp(n int) {
	fmt.Fprintf(os.Stderr, "\033[%dF", n)
}

func getWinsize() (*unix.Winsize, error) {

	ws, err := unix.IoctlGetWinsize(int(os.Stderr.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return nil, os.NewSyscallError("GetWinsize", err)
	}

	return ws, nil
}

func getWidth() int {
	size, err := getWinsize()
	if err != nil {
		return -1
	}
	return int(size.Col)
}

func getHeight() int {
	size, err := getWinsize()
	if err != nil {
		return -1
	}
	return int(size.Row)
}

// strip regexp.
var strip = regexp.MustCompile(`\x1B\[[0-?]*[ -/]*[@-~]`)

func Strip(s string) string {
	return strip.ReplaceAllString(s, "")
}

// Length of characters with ansi escape sequences stripped.
func Length(s string) (n int) {
	for range Strip(s) {
		n++
	}
	return
}
