package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	pbar "github.com/tj/go-progress"
	spin "github.com/tj/go-spin"
	"github.com/tredoe/term"
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
	cursorPos   int
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
	cursorPos   int
}

// Update updates progress with current value
func (p *InfiniteProgress) Update(v TransferStats) {
	p.currentStat = v
}

// String returns a tty representation of current progres
func (p *InfiniteProgress) String() string {
	MoveTo(0, p.cursorPos)
	return fmt.Sprintf("\033[36m\033[m %s transfering: %s, %s, %s",
		p.spinner.Next(),
		byteCountBinary(p.currentStat.transferredBytes),
		fmtDuration(p.currentStat.elapsedTime),
		fmtAvgSpeed(p.currentStat))
}

func fmtAvgSpeed(tf TransferStats) string {
	var speed float64
	if tf.elapsedTime > 0 {
		speed = float64(tf.transferredBytes) / tf.elapsedTime.Seconds()
	}

	return fmt.Sprintf("%s/s", byteCountBinary(int64(speed)))
}

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// NewProgress returns a finite progress indicator if totalSize > 0 otherwise, an infinite progress indicator
func NewProgress(totalSize int64) Progress {

	var progress Progress
	if totalSize < 0 {
		b := pbar.NewInt(int(totalSize))
		b.Width = getWidth() - 20
		progress = &FiniteProgress{
			progressBar: b,
			size:        totalSize,
			currentStat: TransferStats{},
		}
	} else {
		HideCursor()
		lock()
		_, line, _ := GetCursorPosition()
		fmt.Fprintf(os.Stderr, "\n")
		debug("Got position")

		progress = &InfiniteProgress{
			spinner:     spin.New(),
			currentStat: TransferStats{},
			cursorPos:   line,
		}
		releaseLock()
		debug("released lock")
	}

	return progress
}

func lock() {
	flockT := syscall.Flock_t{
		Type:   syscall.F_WRLCK,
		Whence: io.SeekStart,
		Start:  0,
		Len:    1,
	}
	for syscall.FcntlFlock(os.Stderr.Fd(), syscall.F_SETLK, &flockT) != nil {
		time.Sleep(50)
		debug("Could not acquire lock sleeping")
	}

	debug("Got lock")
}

func releaseLock() {
	flockT := syscall.Flock_t{
		Type:   syscall.F_UNLCK,
		Whence: io.SeekStart,
		Start:  0,
		Len:    1,
	}

	err := syscall.FcntlFlock(os.Stderr.Fd(), syscall.F_SETLK, &flockT)
	if err != nil {
		debug("Could not release lock")
	}
}

// MoveTo moves the cursor to (x, y).
func MoveTo(x, y int) {
	fmt.Fprintf(os.Stderr, "\033[%d;%df", y, x)
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
	RestoreCursorPosition()
	return r("\n", ypad) + r(" ", xpad) + s + r("\n", ypad)

}

// SaveCursorPosition saves the cursor position.
func SaveCursorPosition() {
	fmt.Fprintf(os.Stderr, "\033[s")
}

// RestoreCursorPosition saves the cursor position.
func RestoreCursorPosition() {
	fmt.Fprintf(os.Stderr, "\033[u")
}

func MoveUp(n int) {
	fmt.Fprintf(os.Stderr, "\033[%dF", n)
}

// MoveDown moves the cursor to the beginning of n lines down.
func MoveDown(n int) {
	fmt.Fprintf(os.Stderr, "\033[%dE", n)
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

func GetCursorPosition() (col int, line int, err error) {
	// set terminal to raw mode and back
	t, err := term.New()
	if err != nil {
		fallback_SetRawMode()
		defer fallback_SetCookedMode()
	} else {
		t.RawMode()
		defer t.Restore()
	}

	// same as $ echo -e "\033[6n"
	// by printing the output, we are triggering input
	fmt.Fprintf(os.Stderr, fmt.Sprintf("\r%c[6n", 27))

	// capture keyboard output from print command
	reader := bufio.NewReader(os.Stderr)

	// capture the triggered stdin from the print
	text, _ := reader.ReadSlice('R')

	// check for the desired output
	re := regexp.MustCompile(`\d+;\d+`)
	res := re.FindString(string(text))

	// make sure that cooked mode gets set
	if res != "" {
		parts := strings.Split(res, ";")
		line, _ = strconv.Atoi(parts[0])
		col, _ = strconv.Atoi(parts[1])
		return col, line, nil

	} else {
		return 0, 0, errors.New("unable to read cursor position")
	}
}

func fallback_SetRawMode() {
	rawMode := exec.Command("/bin/stty", "raw")
	rawMode.Stdin = os.Stderr
	_ = rawMode.Run()
	rawMode.Wait()
}

func fallback_SetCookedMode() {
	// I've noticed that this does not always work when called from
	// inside the program. From command line, you can run the following
	// '$ go run calling_app.go; stty -raw'
	// if you lose the ability to visably enter new text
	cookedMode := exec.Command("/bin/stty", "-raw")
	cookedMode.Stdin = os.Stderr
	_ = cookedMode.Run()
	cookedMode.Wait()
}

// HideCursor hides the cursor.
func HideCursor() {
	fmt.Fprintf(os.Stderr, "\033[?25l")
}

// ShowCursor shows the cursor.
func ShowCursor() {
	fmt.Fprintf(os.Stderr, "\033[?25h")
}
