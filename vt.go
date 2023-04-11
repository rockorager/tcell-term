package tcellterm

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

type (
	column int
	row    int
)

// VT models a virtual terminal
type VT struct {
	mu sync.Mutex

	activeScreen  [][]cell
	altScreen     [][]cell
	primaryScreen [][]cell

	charset charset
	cursor  cursor
	margin  margin
	mode    mode
	sShift  charset
	tabStop []column

	g0 charset
	g1 charset
	g2 charset
	g3 charset

	savedCursor cursor
	savedDECAWM bool
	savedDECOM  bool

	cmd          *exec.Cmd
	dirty        bool
	eventHandler func(tcell.Event)
	parser       *Parser
	pty          *os.File
	surface      Surface

	step chan bool
}

type margin struct {
	top    row
	bottom row
	left   column
	right  column
}

type charset int

const (
	ascii charset = iota
	decSpecialAndLineDrawing
)

func New() *VT {
	f, _ := os.Create("recording.log")
	log.SetOutput(f)
	return &VT{
		step: make(chan bool),
	}
}

// row, col, style, vis
func (vt *VT) Cursor() (int, int, tcell.CursorStyle, bool) {
	vis := vt.mode&dectcem > 0
	return int(vt.cursor.row), int(vt.cursor.col), vt.cursor.style, vis
}

func (vt *VT) update(seq Sequence) {
	switch seq := seq.(type) {
	case Print:
		vt.print(rune(seq))
	case C0:
		vt.c0(rune(seq))
	case ESC:
		esc := append(seq.Intermediate, seq.Final)
		vt.esc(string(esc))
	case CSI:
		csi := append(seq.Intermediate, seq.Final)
		vt.csi(string(csi), seq.Parameters)
	case OSC:
		vt.osc(string(seq.Payload))
	case DCS:
	case DCSData:
	case DCSEndOfData:
	}
	log.Printf("%s\n", seq)
	// TODO optimize when we post EventRedraw
	vt.postEvent(&EventRedraw{
		EventTerminal: newEventTerminal(vt),
	})
}

func (vt *VT) String() string {
	str := strings.Builder{}
	for row := range vt.activeScreen {
		for col := range vt.activeScreen[row] {
			_, _ = str.WriteRune(vt.activeScreen[row][col].rune())
		}
		if row < vt.height()-1 {
			str.WriteRune('\n')
		}
	}
	return str.String()
}

func (vt *VT) Resize(w int, h int) {
	vt.altScreen = make([][]cell, h)
	vt.primaryScreen = make([][]cell, h)
	for i := range vt.altScreen {
		vt.altScreen[i] = make([]cell, w)
		vt.primaryScreen[i] = make([]cell, w)
	}
	vt.cursor.col = 0
	vt.cursor.row = 0
	vt.margin.top = 0
	vt.margin.bottom = row(h) - 1
	vt.margin.left = 0
	vt.margin.right = column(w) - 1
	switch vt.mode & smcup {
	case 0:
		vt.activeScreen = vt.primaryScreen
	default:
		vt.activeScreen = vt.altScreen
	}

	_ = pty.Setsize(vt.pty, &pty.Winsize{
		Cols: uint16(w),
		Rows: uint16(h),
	})
}

func (vt *VT) width() int {
	if len(vt.activeScreen) > 0 {
		return len(vt.activeScreen[0])
	}
	return 0
}

func (vt *VT) height() int {
	return len(vt.activeScreen)
}

// print sets the current cell contents to the given rune. The attributes will
// be copied from the current cursor attributes
func (vt *VT) print(r rune) {
	if vt.charset == decSpecialAndLineDrawing {
		r = decSpecial[r]
	}

	// If we are single-shifted, move the previous charset into the current
	vt.charset = vt.sShift

	col, row := vt.cursor.position()
	w := column(runewidth.RuneWidth(r))

	if vt.mode&irm != 0 {
		line := vt.activeScreen[row]
		for i := vt.margin.right; i > col; i -= 1 {
			line[i] = line[i-w]
		}
	}

	vt.activeScreen[row][col].content = r
	vt.activeScreen[row][col].attrs = vt.cursor.attrs

	// Set trailing cells to a space if wide rune
	for i := column(1); i < w; i += 1 {
		if col+i > vt.margin.right {
			break
		}
		vt.activeScreen[row][col+i].content = ' '
		vt.activeScreen[row][col+i].attrs = vt.cursor.attrs
	}

	switch {
	case vt.mode&decawm != 0 && col == vt.margin.right:
		vt.nel()
	case col == vt.margin.right:
		// don't move the cursor
	default:
		vt.cursor.col += w
	}
}

// scrollUp shifts all text upward by n rows. Semantically, this is backwards -
// usually scroll up would mean you shift rows down
func (vt *VT) scrollUp(n int) {
	for row := range vt.activeScreen {
		if row > int(vt.margin.bottom) {
			continue
		}
		if row < int(vt.margin.top) {
			continue
		}
		if row+n >= len(vt.activeScreen) {
			vt.activeScreen[row] = make([]cell, len(vt.activeScreen[row]))
			continue
		}
		copy(vt.activeScreen[row], vt.activeScreen[row+n])
	}
}

// scrollDown shifts all lines down by n rows.
func (vt *VT) scrollDown(n int) {
	for row := len(vt.activeScreen) - 1; row >= 0; row -= 1 {
		if row > int(vt.margin.bottom) {
			continue
		}
		if row < int(vt.margin.top) {
			continue
		}
		if row-n < 0 {
			vt.activeScreen[row] = make([]cell, len(vt.activeScreen[row]))
			continue
		}
		copy(vt.activeScreen[row], vt.activeScreen[row-n])
	}
}

// Start starts the terminal with the specified command. Start returns when the
// command has been successfully started.
func (vt *VT) Start(cmd *exec.Cmd) error {
	if cmd == nil {
		return fmt.Errorf("no command to run")
	}
	vt.cmd = cmd
	vt.mu.Lock()
	w, h := vt.surface.Size()
	vt.mu.Unlock()

	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// Start the command with a pty.
	var err error
	winsize := pty.Winsize{
		Cols: uint16(w),
		Rows: uint16(h),
	}
	vt.mu.Lock()
	vt.pty, err = pty.StartWithAttrs(
		cmd,
		&winsize,
		&syscall.SysProcAttr{
			Setsid:  true,
			Setctty: true,
			Ctty:    1,
		})
	vt.mu.Unlock()
	if err != nil {
		return err
	}

	vt.Resize(w, h)
	vt.parser = NewParser(vt.pty)
	go func() {
		for {
			// <-vt.step
			seq := vt.parser.Next()
			switch seq := seq.(type) {
			case EOF:
				return
			default:
				vt.update(seq)
			}
		}
	}()
	return nil
}

func (vt *VT) Close() {
	vt.pty.Close()
}

func (vt *VT) Attach(fn func(ev tcell.Event)) {
	vt.eventHandler = fn
}

func (vt *VT) Detach() {
	vt.eventHandler = func(ev tcell.Event) {
		return
	}
}

func (vt *VT) postEvent(ev tcell.Event) {
	vt.eventHandler(ev)
}

func (vt *VT) SetSurface(srf Surface) {
	vt.surface = srf
}

func (vt *VT) Draw() {
	if vt.surface == nil {
		return
	}
	for row := 0; row < vt.height(); row += 1 {
		for col := 0; col < vt.width(); {
			cell := vt.activeScreen[row][col]
			width := runewidth.RuneWidth(cell.content)
			vt.surface.SetContent(col, row, cell.content, nil, cell.attrs)
			if width == 0 {
				width = 1
			}
			col += width
		}
	}
	// for _, s := range buf.getVisibleSixels() {
	// 	fmt.Printf("\033[%d;%dH", s.Sixel.Y, s.Sixel.X)
	// 	// DECSIXEL Introducer(\033P0;0;8q) + DECGRA ("1;1): Set Raster Attributes
	// 	os.Stdout.Write([]byte{0x1b, 0x50, 0x30, 0x3b, 0x30, 0x3b, 0x38, 0x71, 0x22, 0x31, 0x3b, 0x31})
	// 	os.Stdout.Write(s.Sixel.Data)
	// 	// string terminator(ST)
	// 	os.Stdout.Write([]byte{0x1b, 0x5c})
	// }
}

func (vt *VT) HandleEvent(e tcell.Event) bool {
	switch e := e.(type) {
	case *tcell.EventKey:
		if e.Key() == tcell.KeyEnter {
			// vt.step <- true
		}
		vt.pty.WriteString(keyCode(e))
		return true
	case *tcell.EventPaste:
		switch {
		case vt.mode&paste == 0:
			return false
		case e.Start():
			vt.pty.WriteString(info.PasteStart)
			return true
		case e.End():
			vt.pty.WriteString(info.PasteEnd)
			return true
		}
	case *tcell.EventMouse:
		// if e.Buttons() == tcell.ButtonNone && vt.mouseBtnIn != tcell.ButtonNone {
		// 	// Button was in, and now it's not
		// 	x, y := e.Position()
		// 	s := fmt.Sprintf("\x1b[<%d;%d;%dm", vt.mouseBtnIn-1, x+1, y+1)
		// 	// vt.mouseBtnIn = tcell.ButtonNone
		// 	vt.pty.Write([]byte(s))
		// }
		// if e.Buttons() != tcell.ButtonNone {
		// 	// tcell button map is 1 off from the ansi codes
		// 	// button 0 is main, etc
		// 	btn := e.Buttons() - 1
		// 	x, y := e.Position()
		// 	if vt.mouseBtnIn != tcell.ButtonNone {
		// 		// we are dragging, add 32 to button
		// 		btn = btn + 32
		// 	}
		// 	s := fmt.Sprintf("\x1b[<%d;%d;%dM", btn, x+1, y+1)
		// 	vt.mouseBtnIn = e.Buttons()
		// 	vt.writeToPty([]byte(s))
		// } else {
		// 	vt.mouseBtnIn = tcell.ButtonNone
		// }
	}
	return false
}
