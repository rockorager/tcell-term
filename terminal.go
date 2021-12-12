package main

import (
	"image/color"
	"log"
	"os/exec"

	"git.sr.ht/~ghost08/tcell-term/termutil"
	"github.com/gdamore/tcell/v2"
)

type Terminal struct {
	term *termutil.Terminal
}

func New(opts ...Option) *Terminal {
	t := &Terminal{
		term: termutil.New(),
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

func (t *Terminal) Run(cmd *exec.Cmd, redrawChan chan struct{}, width, height uint16) error {
	return t.term.Run(cmd, redrawChan, height, width)
}

func (t *Terminal) Event(e tcell.Event) {
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
	}
}

func (t *Terminal) Draw(s tcell.Screen, X, Y uint16) {
	buf := t.term.GetActiveBuffer()
	for viewY := int(buf.ViewHeight()) - 1; viewY >= 0; viewY-- {
		for viewX := uint16(0); viewX < buf.ViewWidth(); viewX++ {
			cell := buf.GetCell(viewX, uint16(viewY))
			if cell == nil {
				//s.SetContent(int(viewX+X), viewY+int(Y), ' ', nil, tcell.StyleDefault.Background(tcell.ColorBlack))
				continue
			}
			s.SetContent(int(viewX+X), viewY+int(Y), cell.Rune().Rune, nil, cell.Style())
		}
	}
	if buf.IsCursorVisible() {
		s.ShowCursor(int(buf.CursorColumn()+X), int(buf.CursorLine()+Y))
	} else {
		s.HideCursor()
	}
}

func convertColor(c color.Color, defaultColor tcell.Color) tcell.Color {
	if c == nil {
		return defaultColor
	}
	r, g, b, _ := c.RGBA()
	return tcell.NewRGBColor(int32(r), int32(g), int32(b))
}

type windowManipulator struct{}

func (w *windowManipulator) State() termutil.WindowState {
	log.Println("State")
	return termutil.StateNormal
}
func (w *windowManipulator) Minimise()                { log.Println("Minimise") }
func (w *windowManipulator) Maximise()                { log.Println("Maximise") }
func (w *windowManipulator) Restore()                 { log.Println("Restore") }
func (w *windowManipulator) SetTitle(title string)    { log.Println("SetTitle") }
func (w *windowManipulator) Position() (int, int)     { log.Println("Position"); return 0, 0 }
func (w *windowManipulator) SizeInPixels() (int, int) { log.Println("SizeInPixels"); return 1000, 1000 }
func (w *windowManipulator) CellSizeInPixels() (int, int) {
	log.Println("CellSizeInPixels")
	return 10, 10
}
func (w *windowManipulator) SizeInChars() (int, int) { log.Println("SizeInChars"); return 1000, 1000 }
func (w *windowManipulator) ResizeInPixels(int, int) { log.Println("ResizeInPixels") }
func (w *windowManipulator) ResizeInChars(int, int)  { log.Println("ResizeInChars") }
func (w *windowManipulator) ScreenSizeInPixels() (int, int) {
	log.Println("ScreenSizeInPixels")
	return 1000, 1000
}
func (w *windowManipulator) ScreenSizeInChars() (int, int) {
	log.Println("ScreenSizeInChars")
	return 1000, 1000
}
func (w *windowManipulator) Move(x, y int)              { log.Println("Move") }
func (w *windowManipulator) IsFullscreen() bool         { log.Println("IsFullscreen"); return false }
func (w *windowManipulator) SetFullscreen(enabled bool) { log.Println("SetFullscreen") }
func (w *windowManipulator) GetTitle() string           { log.Println("GetTitle"); return "term" }
func (w *windowManipulator) SaveTitleToStack()          { log.Println("SaveTitleToStack") }
func (w *windowManipulator) RestoreTitleFromStack()     { log.Println("RestoreTitleFromStack") }
func (w *windowManipulator) ReportError(err error)      { log.Println("ReportError") }

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
