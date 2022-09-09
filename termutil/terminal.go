package termutil

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/term"
)

const (
	MainBuffer     uint8 = 0
	AltBuffer      uint8 = 1
)

// Terminal communicates with the underlying terminal
type Terminal struct {
	pty          *os.File
	processChan  chan MeasuredRune
	buffers      []*Buffer
	activeBuffer *Buffer
	mouseMode    MouseMode
	mouseExtMode MouseExtMode
	theme        *Theme
	redraw       bool
	eventCh      chan tcell.Event
	title        string
}

// NewTerminal creates a new terminal instance
func New() *Terminal {
	term := &Terminal{
		processChan: make(chan MeasuredRune, 0xffff),
		theme:       &Theme{},
	}
	fg := term.theme.DefaultForeground()
	bg := term.theme.DefaultBackground()
	term.buffers = []*Buffer{
		NewBuffer(1, 1, 0xffff, fg, bg),
		NewBuffer(1, 1, 0xffff, fg, bg),
	}
	term.activeBuffer = term.buffers[0]
	return term
}

func (t *Terminal) reset() {
	fg := t.theme.DefaultForeground()
	bg := t.theme.DefaultBackground()
	t.buffers = []*Buffer{
		NewBuffer(1, 1, 0xffff, fg, bg),
		NewBuffer(1, 1, 0xffff, fg, bg),
		NewBuffer(1, 1, 0xffff, fg, bg),
	}
	t.useMainBuffer()
}

// Pty exposes the underlying terminal pty, if it exists
func (t *Terminal) Pty() *os.File {
	return t.pty
}

func (t *Terminal) WriteToPty(data []byte) error {
	_, err := t.pty.Write(data)
	return err
}

func (t *Terminal) GetTitle() string {
	return t.title
}

func (t *Terminal) Theme() *Theme {
	return t.theme
}

// write takes data from StdOut of the child shell and processes it
func (t *Terminal) Write(data []byte) (n int, err error) {
	reader := bufio.NewReader(bytes.NewBuffer(data))
	for {
		r, size, err := reader.ReadRune()
		if err == io.EOF {
			break
		}
		t.processChan <- MeasuredRune{Rune: r, Width: size}
	}
	return len(data), nil
}

func (t *Terminal) SetSize(rows, cols uint16) error {
	if t.pty == nil {
		return fmt.Errorf("terminal is not running")
	}

	t.activeBuffer.resizeView(cols, rows)

	if err := pty.Setsize(t.pty, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	}); err != nil {
		return err
	}

	return nil
}

// Run starts the terminal/shell proxying process
func (t *Terminal) Run(c *exec.Cmd, rows uint16, cols uint16, attr *syscall.SysProcAttr, eventCh chan tcell.Event) error {
	c.Env = append(os.Environ(), "TERM=xterm-256color")

	t.eventCh = eventCh
	// Start the command with a pty.
	var err error
	// t.pty, err = pty.Start(c)
	winsize := pty.Winsize{
		Cols: cols,
		Rows: rows,
	}
	t.pty, err = pty.StartWithAttrs(c, &winsize, &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 1})
	if err != nil {
		return err
	}
	// Make sure to close the pty at the end.
	defer func() { _ = t.pty.Close() }() // Best effort.

	if err := t.SetSize(rows, cols); err != nil {
		return err
	}

	// Set stdin in raw mode.
	if fd := int(os.Stdin.Fd()); term.IsTerminal(fd) {
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			// TODO send an event?
		}
		defer func() { _ = term.Restore(fd, oldState) }() // Best effort.
	}

	go t.process()

	_, _ = io.Copy(t, t.pty)
	return nil
}

func (t *Terminal) ShouldRedraw() bool {
	return t.redraw
}

func (t *Terminal) SetRedraw(b bool) {
	t.redraw = b
}

func (t *Terminal) process() {
	for {
		mr, ok := <-t.processChan
		if !ok {
			return
		}
		if mr.Rune == 0x1b { // ANSI escape char, which means this is a sequence
			if t.handleANSI(t.processChan) {
				t.SetRedraw(true)
			}
		} else if t.processRunes(mr) { // otherwise it's just an individual rune we need to process
			t.SetRedraw(true)
		}
	}
}

func (t *Terminal) processRunes(runes ...MeasuredRune) (renderRequired bool) {
	for _, r := range runes {
		switch r.Rune {
		case 0x05: // enq
			continue
		case 0x07: // bell
			// DING DING DING
			continue
		case 0x8: // backspace
			t.GetActiveBuffer().backspace()
			renderRequired = true
		case 0x9: // tab
			t.GetActiveBuffer().tab()
			renderRequired = true
		case 0xa, 0xc: // newLine/form feed
			t.GetActiveBuffer().newLine()
			renderRequired = true
		case 0xb: // vertical tab
			t.GetActiveBuffer().verticalTab()
			renderRequired = true
		case 0xd: // carriageReturn
			t.GetActiveBuffer().carriageReturn()
			renderRequired = true
		case 0xe: // shiftOut
			t.GetActiveBuffer().currentCharset = 1
		case 0xf: // shiftIn
			t.GetActiveBuffer().currentCharset = 0
		default:
			if r.Rune < 0x20 {
				// handle any other control chars here?
				continue
			}

			t.GetActiveBuffer().write(t.translateRune(r))
			renderRequired = true
		}
	}

	return renderRequired
}

func (t *Terminal) translateRune(b MeasuredRune) MeasuredRune {
	table := t.GetActiveBuffer().charsets[t.GetActiveBuffer().currentCharset]
	if table == nil {
		return b
	}
	chr, ok := (*table)[b.Rune]
	if ok {
		return MeasuredRune{Rune: chr, Width: 1}
	}
	return b
}

func (t *Terminal) setTitle(title string) {
	t.title = title
	t.eventCh <- newEventTitle(title)
}

func (t *Terminal) switchBuffer(index uint8) {
	var carrySize bool
	var w, h uint16
	if t.activeBuffer != nil {
		w, h = t.activeBuffer.viewWidth, t.activeBuffer.viewHeight
		carrySize = true
	}
	t.activeBuffer = t.buffers[index]
	if carrySize {
		t.activeBuffer.resizeView(w, h)
	}
}

func (t *Terminal) GetMouseMode() MouseMode {
	return t.mouseMode
}

func (t *Terminal) GetMouseExtMode() MouseExtMode {
	return t.mouseExtMode
}

func (t *Terminal) GetActiveBuffer() *Buffer {
	return t.activeBuffer
}

func (t *Terminal) useMainBuffer() {
	t.switchBuffer(MainBuffer)
}

func (t *Terminal) useAltBuffer() {
	t.switchBuffer(AltBuffer)
}

type EventTitle struct {
	when  time.Time
	title string
}

func (ev *EventTitle) When() time.Time {
	return ev.when
}

func (ev *EventTitle) Title() string {
	return ev.title
}

func newEventTitle(title string) tcell.Event {
	return &EventTitle{
		when:  time.Now(),
		title: title,
	}
}
