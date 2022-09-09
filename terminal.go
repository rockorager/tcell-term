package tcellterm

import (
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"syscall"
	"time"

	"git.sr.ht/~rockorager/tcell-term/termutil"
	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"
)

type Terminal struct {
	term *termutil.Terminal

	curX     int
	curY     int
	curStyle tcell.CursorStyle
	curVis   bool

	view     views.View
	interval int

	close bool

	views.WidgetWatchers
}

func New(opts ...Option) *Terminal {
	t := &Terminal{
		term:     termutil.New(),
		interval: 8,
	}
	t.term.SetWindowManipulator(&windowManipulator{})
	for _, opt := range opts {
		opt(t)
	}
	return t
}

type Option func(*Terminal)

func WithWindowManipulator(wm termutil.WindowManipulator) Option {
	return func(t *Terminal) {
		t.term.SetWindowManipulator(wm)
	}
}

// WithPollInterval sets the minimum time, in ms, between
// views.EventWidgetContent events, which signal the screen has updates which
// can be drawn.
//
// Default: 8 ms
func WithPollInterval(interval int) Option {
	return func(t *Terminal) {
		if interval < 1 {
			interval = 1
		}
		t.interval = interval
	}
}

func (t *Terminal) Run(cmd *exec.Cmd) error {
	return t.run(cmd, &syscall.SysProcAttr{})
}

func (t *Terminal) RunWithAttrs(cmd *exec.Cmd, attr *syscall.SysProcAttr) error {
	return t.run(cmd, attr)
}

func (t *Terminal) run(cmd *exec.Cmd, attr *syscall.SysProcAttr) error {
	w, h := t.view.Size()
	tmr := time.NewTicker(time.Duration(t.interval) * time.Millisecond)
	eventCh := make(chan tcell.Event)
	go func() {
		for {
			select {
			case <-tmr.C:
				if t.close {
					return
				}
				if t.term.ShouldRedraw() {
					t.PostEventWidgetContent(t)
					t.term.SetRedraw(false)
				}
			case ev := <-eventCh:
				switch ev := ev.(type) {
				case *termutil.EventTitle:
					t.PostEvent(&EventTitle{
						widget: t,
						when:   ev.When(),
						title:  ev.Title(),
					})
				}
			}
		}
	}()

	err := t.term.Run(cmd, uint16(h), uint16(w), attr, eventCh)
	if err != nil {
		return err
	}
	t.Close()
	return nil
}

type EventTitle struct {
	when   time.Time
	title  string
	widget *Terminal
}

func (ev *EventTitle) When() time.Time {
	return ev.when
}

func (ev *EventTitle) Widget() views.Widget {
	return ev.widget
}

func (ev *EventTitle) Title() string {
	return ev.title
}

func (t *Terminal) Close() {
	t.close = true
	t.term.Pty().Close()
}

// SetView sets the view for the terminal to draw to. This must be set before
// calling Draw
func (t *Terminal) SetView(view views.View) {
	t.view = view
}

func (t *Terminal) Size() (int, int) {
	if t.view == nil {
		return 0, 0
	}
	return t.view.Size()
}

func (t *Terminal) HandleEvent(e tcell.Event) bool {
	switch e := e.(type) {
	case *tcell.EventKey:
		var keycode string
		switch {
		case e.Modifiers()&tcell.ModCtrl != 0:
			keycode = getCtrlCombinationKeyCode(e)
		case e.Modifiers()&tcell.ModAlt != 0:
			keycode = getAltCombinationKeyCode(e)
		default:
			keycode = getKeyCode(e)
		}
		t.term.WriteToPty([]byte(keycode))
		return true
	}
	return false
}

func (t *Terminal) Draw() {
	if t.view == nil {
		return
	}
	buf := t.term.GetActiveBuffer()
	w, h := t.view.Size()
	for viewY := 0; viewY < h; viewY++ {
		for viewX := uint16(0); viewX < uint16(w); viewX++ {
			cell := buf.GetCell(viewX, uint16(viewY))
			if cell == nil {
				t.view.SetContent(int(viewX), viewY, ' ', nil, tcell.StyleDefault)
			} else if cell.Dirty() {
				t.view.SetContent(int(viewX), viewY, cell.Rune().Rune, nil, cell.Style())
			}
		}
	}
	if buf.IsCursorVisible() {
		t.curVis = true
		t.curX = int(buf.CursorColumn())
		t.curY = int(buf.CursorLine())
		t.curStyle = tcell.CursorStyle(t.term.GetActiveBuffer().GetCursorShape())
	} else {
		t.curVis = false
	}
	for _, s := range buf.GetVisibleSixels() {
		fmt.Printf("\033[%d;%dH", s.Sixel.Y, s.Sixel.X)
		// DECSIXEL Introducer(\033P0;0;8q) + DECGRA ("1;1): Set Raster Attributes
		os.Stdout.Write([]byte{0x1b, 0x50, 0x30, 0x3b, 0x30, 0x3b, 0x38, 0x71, 0x22, 0x31, 0x3b, 0x31})
		os.Stdout.Write(s.Sixel.Data)
		// string terminator(ST)
		os.Stdout.Write([]byte{0x1b, 0x5c})
	}
}

// GetCursor returns if the cursor is visible, it's x and y position, and it's
// style. If the cursor is not visible, the coordinates will be -1,-1
func (t *Terminal) GetCursor() (bool, int, int, tcell.CursorStyle) {
	return t.curVis, t.curX, t.curY, t.curStyle
}

func convertColor(c color.Color, defaultColor tcell.Color) tcell.Color {
	if c == nil {
		return defaultColor
	}
	r, g, b, _ := c.RGBA()
	return tcell.NewRGBColor(int32(r), int32(g), int32(b))
}

// Resize resizes the terminal to the dimensions of the terminals view
func (t *Terminal) Resize() {
	if t.view == nil {
		return
	}
	w, h := t.view.Size()
	t.term.SetSize(uint16(h), uint16(w))
}

type windowManipulator struct{}

func (w *windowManipulator) State() termutil.WindowState {
	return termutil.StateNormal
}
func (w *windowManipulator) Minimise()             {}
func (w *windowManipulator) Maximise()             {}
func (w *windowManipulator) Restore()              {}
func (w *windowManipulator) SetTitle(title string) {}
func (w *windowManipulator) Position() (int, int)  { return 0, 0 }
func (w *windowManipulator) SizeInPixels() (int, int) {
	sz, _ := GetWinSize()
	return int(sz.XPixel), int(sz.YPixel)
}

func (w *windowManipulator) CellSizeInPixels() (int, int) {
	sz, _ := GetWinSize()
	return int(sz.Cols / sz.XPixel), int(sz.Rows / sz.YPixel)
}

func (w *windowManipulator) SizeInChars() (int, int) {
	sz, _ := GetWinSize()
	return int(sz.Cols), int(sz.Rows)
}
func (w *windowManipulator) ResizeInPixels(int, int) {}
func (w *windowManipulator) ResizeInChars(int, int)  {}
func (w *windowManipulator) ScreenSizeInPixels() (int, int) {
	return w.SizeInPixels()
}

func (w *windowManipulator) ScreenSizeInChars() (int, int) {
	return w.SizeInChars()
}
func (w *windowManipulator) Move(x, y int)              {}
func (w *windowManipulator) IsFullscreen() bool         { return false }
func (w *windowManipulator) SetFullscreen(enabled bool) {}
func (w *windowManipulator) GetTitle() string           { return "term" }
func (w *windowManipulator) SaveTitleToStack()          {}
func (w *windowManipulator) RestoreTitleFromStack()     {}
func (w *windowManipulator) ReportError(err error)      {}

func getCtrlCombinationKeyCode(ke *tcell.EventKey) string {
	if keycode, ok := LINUX_CTRL_KEY_MAP[ke.Key()]; ok {
		return keycode
	}
	if keycode, ok := LINUX_CTRL_RUNE_MAP[ke.Rune()]; ok {
		return keycode
	}
	if ke.Key() == tcell.KeyRune {
		r := ke.Rune()
		if r >= 97 && r <= 122 {
			r = r - 'a' + 1
			return string(r)
		}
	}
	return getKeyCode(ke)
}

func getAltCombinationKeyCode(ke *tcell.EventKey) string {
	if keycode, ok := LINUX_ALT_KEY_MAP[ke.Key()]; ok {
		return keycode
	}
	code := getKeyCode(ke)
	return "\x1b" + code
}

func getKeyCode(ke *tcell.EventKey) string {
	if keycode, ok := LINUX_KEY_MAP[ke.Key()]; ok {
		return keycode
	}
	return string(ke.Rune())
}

var (
	LINUX_KEY_MAP = map[tcell.Key]string{
		tcell.KeyEnter:      "\r",
		tcell.KeyBackspace:  "\x7f",
		tcell.KeyBackspace2: "\x7f",
		tcell.KeyTab:        "\t",
		tcell.KeyEscape:     "\x1b",
		tcell.KeyDown:       "\x1b[B",
		tcell.KeyUp:         "\x1b[A",
		tcell.KeyRight:      "\x1b[C",
		tcell.KeyLeft:       "\x1b[D",
		tcell.KeyHome:       "\x1b[1~",
		tcell.KeyEnd:        "\x1b[4~",
		tcell.KeyPgUp:       "\x1b[5~",
		tcell.KeyPgDn:       "\x1b[6~",
		tcell.KeyDelete:     "\x1b[3~",
		tcell.KeyInsert:     "\x1b[2~",
		tcell.KeyF1:         "\x1bOP",
		tcell.KeyF2:         "\x1bOQ",
		tcell.KeyF3:         "\x1bOR",
		tcell.KeyF4:         "\x1bOS",
		tcell.KeyF5:         "\x1b[15~",
		tcell.KeyF6:         "\x1b[17~",
		tcell.KeyF7:         "\x1b[18~",
		tcell.KeyF8:         "\x1b[19~",
		tcell.KeyF9:         "\x1b[20~",
		tcell.KeyF10:        "\x1b[21~",
		tcell.KeyF12:        "\x1b[24~",
		/*
			"bracketed_paste_mode_start": "\x1b[200~",
			"bracketed_paste_mode_end":   "\x1b[201~",
		*/
	}

	LINUX_CTRL_KEY_MAP = map[tcell.Key]string{
		tcell.KeyUp:    "\x1b[1;5A",
		tcell.KeyDown:  "\x1b[1;5B",
		tcell.KeyRight: "\x1b[1;5C",
		tcell.KeyLeft:  "\x1b[1;5D",
	}

	LINUX_CTRL_RUNE_MAP = map[rune]string{
		'@':  "\x00",
		'`':  "\x00",
		'[':  "\x1b",
		'{':  "\x1b",
		'\\': "\x1c",
		'|':  "\x1c",
		']':  "\x1d",
		'}':  "\x1d",
		'^':  "\x1e",
		'~':  "\x1e",
		'_':  "\x1f",
		'?':  "\x7f",
	}

	LINUX_ALT_KEY_MAP = map[tcell.Key]string{
		tcell.KeyUp:    "\x1b[1;3A",
		tcell.KeyDown:  "\x1b[1;3B",
		tcell.KeyRight: "\x1b[1;3C",
		tcell.KeyLeft:  "\x1b[1;3D",
	}
)
