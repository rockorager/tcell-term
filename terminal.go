package tcellterm

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"
)

const (
	mainBuffer uint8 = 0
	altBuffer  uint8 = 1
)

type Terminal struct {
	curX     int
	curY     int
	curStyle tcell.CursorStyle
	curVis   bool
	view     views.View
	interval int
	close    bool
	views.WidgetWatchers
	pty          *os.File
	processChan  chan measuredRune
	buffers      []*buffer
	activeBuffer *buffer
	mouseBtnEvnt bool
	mouseDrgEvnt bool
	mouseMtnEvnt bool
	mouseBtnIn   tcell.ButtonMask
	recorder     *os.File
	redraw       bool
	title        string
	mu           sync.Mutex
}

func New(opts ...option) *Terminal {
	term := &Terminal{
		processChan: make(chan measuredRune, 0xffff),
		interval:    8,
	}
	fg := defaultForeground()
	bg := defaultBackground()
	term.buffers = []*buffer{
		newBuffer(1, 1, 0xffff, fg, bg),
		newBuffer(1, 1, 0xffff, fg, bg),
	}
	term.activeBuffer = term.buffers[0]
	for _, opt := range opts {
		opt(term)
	}
	return term
}

func (t *Terminal) reset() {
	fg := defaultForeground()
	bg := defaultBackground()
	t.buffers = []*buffer{
		newBuffer(1, 1, 0xffff, fg, bg),
		newBuffer(1, 1, 0xffff, fg, bg),
	}
	t.useMainBuffer()
}

type option func(*Terminal)

// WithPollInterval sets the minimum time, in ms, between
// views.EventWidgetContent events, which signal the screen has updates which
// can be drawn.
//
// Default: 8 ms
func WithPollInterval(interval int) option {
	return func(t *Terminal) {
		if interval < 1 {
			interval = 1
		}
		t.interval = interval
	}
}

func WithRecorder(p string) option {
	return func(t *Terminal) {
		if !path.IsAbs(p) {
			wd, err := os.Getwd()
			if err != nil {
				return
			}
			path.Join(wd, p)
		}
		t.recorder, _ = os.Create(p)
	}
}

// Run starts the terminal with the specified command
func (t *Terminal) Run(cmd *exec.Cmd) error {
	return t.run(cmd, &syscall.SysProcAttr{})
}

// Run starts the terminal with the specified command and custom attributes
func (t *Terminal) RunWithAttrs(cmd *exec.Cmd, attr *syscall.SysProcAttr) error {
	return t.run(cmd, attr)
}

func (t *Terminal) run(cmd *exec.Cmd, attr *syscall.SysProcAttr) error {
	if cmd == nil {
		return fmt.Errorf("no command to run")
	}
	w, h := t.view.Size()
	tmr := time.NewTicker(time.Duration(t.interval) * time.Millisecond)
	go func() {
		for range tmr.C {
			if t.shouldClose() {
				if cmd != nil && cmd.Process != nil {
					cmd.Process.Kill()
					cmd.Wait()
				}
				t.PostEvent(&EventClosed{
					EventTerminal: newEventTerminal(t),
				})
				return
			}
			if t.ShouldRedraw() {
				t.PostEventWidgetContent(t)
			}
		}
	}()

	// Start the command with a pty.
	var err error
	winsize := pty.Winsize{
		Cols: uint16(w),
		Rows: uint16(h),
	}
	t.mu.Lock()
	t.pty, err = pty.StartWithAttrs(
		cmd,
		&winsize,
		&syscall.SysProcAttr{
			Setsid:  true,
			Setctty: true,
			Ctty:    1,
		})
	t.mu.Unlock()
	if err != nil {
		return err
	}
	defer t.pty.Close()

	err = t.setSize(h, w)
	if err != nil {
		return err
	}
	// TODO This is most likely not needed
	// Set stdin in raw mode.
	// if fd := int(os.Stdin.Fd()); term.IsTerminal(fd) {
	// 	oldState, _ := term.MakeRaw(fd)
	// 	defer term.Restore(fd, oldState)
	// }
	go t.process()
	_, _ = io.Copy(t, t.pty)
	t.Close()
	return nil
}

// Close ends the process and cleans up the terminal. An EventClosed event will
// be emitted when the terminal has closed
func (t *Terminal) Close() {
	t.mu.Lock()
	t.close = true
	t.mu.Unlock()
}

func (t *Terminal) shouldClose() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.close
}

// SetView sets the view for the terminal to draw to. This must be set before
// calling Draw. Setting the view also calls Resize(). Any change to the
// underlying view requires the host application to call Resize again.
func (t *Terminal) SetView(view views.View) {
	t.view = view
	t.Resize()
}

// Size reports the current view size in rows, cols
func (t *Terminal) Size() (int, int) {
	if t.view == nil {
		return 0, 0
	}
	return t.view.Size()
}

// HandleEvent handles tcell Events from the parent application
func (t *Terminal) HandleEvent(e tcell.Event) bool {
	switch e := e.(type) {
	case *tcell.EventKey:
		var keycode string
		switch {
		case e.Modifiers()&tcell.ModAlt != 0:
			keycode = getAltCombinationKeyCode(e)
		case e.Modifiers()&tcell.ModCtrl != 0:
			keycode = getCtrlCombinationKeyCode(e)
		default:
			keycode = getKeyCode(e)
		}
		t.writeToPty([]byte(keycode))
		return true
	case *tcell.EventResize:
		t.Resize()
		return true
	case *tcell.EventPaste:
		if e.Start() {
			t.StartPaste()
		}
		if e.End() {
			t.EndPaste()
		}
		return true
	case *tcell.EventMouse:
		if e.Buttons() == tcell.ButtonNone && t.mouseBtnIn != tcell.ButtonNone {
			// Button was in, and now it's not
			x, y := e.Position()
			s := fmt.Sprintf("\x1b[<%d;%d;%dm", t.mouseBtnIn-1, x+1, y+1)
			t.mouseBtnIn = tcell.ButtonNone
			t.writeToPty([]byte(s))
		}
		if e.Buttons() != tcell.ButtonNone {
			// tcell button map is 1 off from the ansi codes
			// button 0 is main, etc
			btn := e.Buttons() - 1
			x, y := e.Position()
			if t.mouseBtnIn != tcell.ButtonNone {
				// we are dragging, add 32 to button
				btn = btn + 32
			}
			s := fmt.Sprintf("\x1b[<%d;%d;%dM", btn, x+1, y+1)
			t.mouseBtnIn = e.Buttons()
			t.writeToPty([]byte(s))
		} else {
			t.mouseBtnIn = tcell.ButtonNone
		}
	}
	return false
}

// Draw draws the current cell buffer to the view.
func (t *Terminal) Draw() {
	if t.view == nil {
		return
	}
	t.mu.Lock()
	t.SetRedraw(false)
	buf := t.getActiveBuffer()
	w, h := t.view.Size()
	for viewY := 0; viewY < h; viewY++ {
		for viewX := 0; viewX < w; viewX++ {
			cell := buf.getCell(viewX, viewY)
			if cell == nil {
				t.view.SetContent(viewX, viewY, ' ', nil, tcell.StyleDefault)
			} else {
				t.view.SetContent(viewX, viewY, cell.rune().rune, nil, cell.style())
			}
		}
	}
	if buf.isCursorVisible() {
		t.curVis = true
		t.curX = int(buf.cursorColumn())
		t.curY = int(buf.cursorLine())
		t.curStyle = tcell.CursorStyle(t.getActiveBuffer().getCursorShape())
	} else {
		t.curVis = false
	}
	for _, s := range buf.getVisibleSixels() {
		fmt.Printf("\033[%d;%dH", s.Sixel.Y, s.Sixel.X)
		// DECSIXEL Introducer(\033P0;0;8q) + DECGRA ("1;1): Set Raster Attributes
		os.Stdout.Write([]byte{0x1b, 0x50, 0x30, 0x3b, 0x30, 0x3b, 0x38, 0x71, 0x22, 0x31, 0x3b, 0x31})
		os.Stdout.Write(s.Sixel.Data)
		// string terminator(ST)
		os.Stdout.Write([]byte{0x1b, 0x5c})
	}
	t.mu.Unlock()
}

// GetCursor returns if the cursor is visible, it's x and y position, and it's
// style. If the cursor is not visible, the coordinates will be -1,-1
func (t *Terminal) GetCursor() (bool, int, int, tcell.CursorStyle) {
	return t.curVis, t.curX, t.curY, t.curStyle
}

// Resize resizes the terminal to the dimensions of the terminals view
func (t *Terminal) Resize() {
	if t.view == nil {
		return
	}
	w, h := t.view.Size()
	t.setSize(h, w)
	t.Draw()
}

func (t *Terminal) getActiveBuffer() *buffer {
	return t.activeBuffer
}

func (t *Terminal) processRunes(runes ...measuredRune) (renderRequired bool) {
	for _, r := range runes {
		switch r.rune {
		case 0x05: // enq
			continue
		case 0x07: // bell
			// DING DING DING
			continue
		case 0x8: // backspace
			t.getActiveBuffer().backspace()
			renderRequired = true
		case 0x9: // tab
			t.getActiveBuffer().tab()
			renderRequired = true
		case 0xa, 0xc: // newLine/form feed
			t.getActiveBuffer().newLine()
			renderRequired = true
		case 0xb: // vertical tab
			t.getActiveBuffer().verticalTab()
			renderRequired = true
		case 0xd: // carriageReturn
			t.getActiveBuffer().carriageReturn()
			renderRequired = true
		case 0xe: // shiftOut
			t.getActiveBuffer().currentCharset = 1
		case 0xf: // shiftIn
			t.getActiveBuffer().currentCharset = 0
		default:
			if r.rune < 0x20 {
				// handle any other control chars here?
				continue
			}

			t.getActiveBuffer().write(t.translateRune(r))
			renderRequired = true
		}
	}

	return renderRequired
}

func (t *Terminal) translateRune(b measuredRune) measuredRune {
	table := t.getActiveBuffer().charsets[t.getActiveBuffer().currentCharset]
	if table == nil {
		return b
	}
	chr, ok := (*table)[b.rune]
	if ok {
		return measuredRune{rune: chr, width: 1}
	}
	return b
}

func (t *Terminal) useAltBuffer() {
	t.switchBuffer(altBuffer)
}

func (t *Terminal) switchBuffer(index uint8) {
	var carrySize bool
	var w, h int
	if t.activeBuffer != nil {
		w, h = t.activeBuffer.viewWidth, t.activeBuffer.viewHeight
		carrySize = true
	}
	t.activeBuffer = t.buffers[index]
	if carrySize {
		t.activeBuffer.resizeView(w, h)
	}
}

func (t *Terminal) useMainBuffer() {
	t.switchBuffer(mainBuffer)
}

func (t *Terminal) setTitle(title string) {
	t.title = title
	t.PostEvent(&EventTitle{
		title:         title,
		EventTerminal: newEventTerminal(t),
	})
}

// ShouldRedraw returns whether any cell in the cell buffer is dirty
func (t *Terminal) ShouldRedraw() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.redraw
}

// SetRedraw sets the dirty state of the cell buffer. The host application
// should set this to false after a draw is performed
func (t *Terminal) SetRedraw(b bool) {
	t.redraw = b
}

func (t *Terminal) setSize(rows, cols int) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.pty == nil {
		return fmt.Errorf("terminal is not running")
	}

	t.activeBuffer.resizeView(cols, rows)

	if err := pty.Setsize(t.pty, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	}); err != nil {
		return err
	}

	return nil
}

func (t *Terminal) process() {
	for {
		mr, ok := <-t.processChan
		if !ok {
			return
		}
		t.mu.Lock()
		if mr.rune == 0x1b { // ANSI escape char, which means this is a sequence
			if t.handleANSI(t.processChan) {
				t.SetRedraw(true)
			}
		} else if t.processRunes(mr) { // otherwise it's just an individual rune we need to process
			t.SetRedraw(true)
		}
		t.mu.Unlock()
	}
}

// Write takes data from StdOut of the child shell and processes it
func (t *Terminal) Write(data []byte) (n int, err error) {
	reader := bufio.NewReader(bytes.NewBuffer(data))
	if t.recorder != nil {
		t.recorder.Write(data)
	}
	for {
		r, size, err := reader.ReadRune()
		if err == io.EOF {
			break
		}
		t.processChan <- measuredRune{rune: r, width: size}
	}
	return len(data), nil
}

func (t *Terminal) writeToPty(data []byte) error {
	_, err := t.pty.Write(data)
	return err
}

// StartPaste begins a bracketed paste segment. This should be called in
// response to a tcell.EventPaste event. Alternatively, the event can be passed
// to the terminal's event handler for proper handling.
func (t *Terminal) StartPaste() {
	if !t.activeBuffer.modes.BracketedPasteMode {
		return
	}
	t.writeToPty([]byte("\x1b[200~"))
}

// EndPaste ends a bracketed paste segment. This should be called in response
// to a tcell.EventPaste event. Alternatively, the event can be passed to the
// terminal's event handler for proper handling.
func (t *Terminal) EndPaste() {
	if !t.activeBuffer.modes.BracketedPasteMode {
		return
	}
	t.writeToPty([]byte("\x1b[201~"))
}
