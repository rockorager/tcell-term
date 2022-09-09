package termutil

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
)

func parseCSI(readChan chan MeasuredRune) (final rune, params []string, intermediate []rune, raw []rune) {
	var b MeasuredRune

	param := ""
	intermediate = []rune{}
CSI:
	for {
		b = <-readChan
		raw = append(raw, b.Rune)
		switch true {
		case b.Rune >= 0x30 && b.Rune <= 0x3F:
			param = param + string(b.Rune)
		case b.Rune > 0 && b.Rune <= 0x2F:
			intermediate = append(intermediate, b.Rune)
		case b.Rune >= 0x40 && b.Rune <= 0x7e:
			final = b.Rune
			break CSI
		}
	}

	unprocessed := []string{}
	if strings.Contains(param, ":") {
		unprocessed = strings.Split(param, ":")
	} else {
		unprocessed = strings.Split(param, ";")
	}
	for _, par := range unprocessed {
		if par != "" {
			par = strings.TrimLeft(par, "0")
			if par == "" {
				par = "0"
			}
			params = append(params, par)
		}
	}

	return final, params, intermediate, raw
}

func (t *Terminal) handleCSI(readChan chan MeasuredRune) (renderRequired bool) {
	final, params, intermediate, raw := parseCSI(readChan)

	switch final {
	case 'c':
		return t.csiSendDeviceAttributesHandler(params)
	case 'd':
		return t.csiLinePositionAbsoluteHandler(params)
	case 'f':
		return t.csiCursorPositionHandler(params)
	case 'g':
		return t.csiTabClearHandler(params)
	case 'h':
		return t.csiSetModeHandler(params)
	case 'l':
		return t.csiResetModeHandler(params)
	case 'm':
		return t.sgrSequenceHandler(params)
	case 'n':
		return t.csiDeviceStatusReportHandler(params)
	case 'r':
		return t.csiSetMarginsHandler(params)
	case 't':
		// Window manipulation, unsupported
		return false
	case 'q':
		if string(intermediate) == " " {
			return t.csiCursorSelection(params)
		}
	case 'A':
		return t.csiCursorUpHandler(params)
	case 'B':
		return t.csiCursorDownHandler(params)
	case 'C':
		return t.csiCursorForwardHandler(params)
	case 'D':
		return t.csiCursorBackwardHandler(params)
	case 'E':
		return t.csiCursorNextLineHandler(params)
	case 'F':
		return t.csiCursorPrecedingLineHandler(params)
	case 'G':
		return t.csiCursorCharacterAbsoluteHandler(params)
	case 'H':
		return t.csiCursorPositionHandler(params)
	case 'J':
		return t.csiEraseInDisplayHandler(params)
	case 'K':
		return t.csiEraseInLineHandler(params)
	case 'L':
		return t.csiInsertLinesHandler(params)
	case 'M':
		return t.csiDeleteLinesHandler(params)
	case 'P':
		return t.csiDeleteHandler(params)
	case 'S':
		return t.csiScrollUpHandler(params)
	case 'T':
		return t.csiScrollDownHandler(params)
	case 'X':
		return t.csiEraseCharactersHandler(params)
	case '@':
		return t.csiInsertBlankCharactersHandler(params)
	case 'p': // reset handler
		if string(intermediate) == "!" {
			return t.csiSoftResetHandler(params)
		}
		return false
	}

	for _, b := range intermediate {
		t.processRunes(MeasuredRune{
			Rune:  b,
			Width: 1,
		})
	}

	// TODO review this:
	// if this is an unknown CSI sequence, write it to stdout as we can't handle it?
	//_ = t.writeToRealStdOut(append([]rune{0x1b, '['}, raw...)...)
	_ = raw
	// log.Printf("UNKNOWN CSI P(%s) I(%s) %c", strings.Join(params, ";"), string(intermediate), final)
	return false
}

// CSI c
// Send Device Attributes (Primary/Secondary/Tertiary DA)
func (t *Terminal) csiSendDeviceAttributesHandler(params []string) (renderRequired bool) {
	// we are VT100
	// for DA1 we'll respond ?1;2
	// for DA2 we'll respond >0;0;0
	response := "?1;2"
	if len(params) > 0 && len(params[0]) > 0 && params[0][0] == '>' {
		response = ">0;0;0"
	}

	// write response to source pty
	t.WriteToPty([]byte("\x1b[" + response + "c"))
	return false
}

// CSI n
// Device Status Report (DSR)
func (t *Terminal) csiDeviceStatusReportHandler(params []string) (renderRequired bool) {
	if len(params) == 0 {
		return false
	}

	switch params[0] {
	case "5":
		t.WriteToPty([]byte("\x1b[0n")) // everything is cool
	case "6": // report cursor position
		t.WriteToPty([]byte(fmt.Sprintf(
			"\x1b[%d;%dR",
			t.GetActiveBuffer().CursorLine()+1,
			t.GetActiveBuffer().CursorColumn()+1,
		)))
	}

	return false
}

// CSI A
// Cursor Up Ps Times (default = 1) (CUU)
func (t *Terminal) csiCursorUpHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}
	t.GetActiveBuffer().movePosition(0, -int16(distance))
	return true
}

// CSI B
// Cursor Down Ps Times (default = 1) (CUD)
func (t *Terminal) csiCursorDownHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}

	t.GetActiveBuffer().movePosition(0, int16(distance))
	return true
}

// CSI C
// Cursor Forward Ps Times (default = 1) (CUF)
func (t *Terminal) csiCursorForwardHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}

	t.GetActiveBuffer().movePosition(int16(distance), 0)
	return true
}

// CSI D
// Cursor Backward Ps Times (default = 1) (CUB)
func (t *Terminal) csiCursorBackwardHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}

	t.GetActiveBuffer().movePosition(-int16(distance), 0)
	return true
}

// CSI E
// Cursor Next Line Ps Times (default = 1) (CNL)
func (t *Terminal) csiCursorNextLineHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}

	t.GetActiveBuffer().movePosition(0, int16(distance))
	t.GetActiveBuffer().setPosition(0, t.GetActiveBuffer().CursorLine())
	return true
}

// CSI F
// Cursor Preceding Line Ps Times (default = 1) (CPL)
func (t *Terminal) csiCursorPrecedingLineHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}
	t.GetActiveBuffer().movePosition(0, -int16(distance))
	t.GetActiveBuffer().setPosition(0, t.GetActiveBuffer().CursorLine())
	return true
}

// CSI G
// Cursor Horizontal Absolute  [column] (default = [row,1]) (CHA)
func (t *Terminal) csiCursorCharacterAbsoluteHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || params[0] == "" {
			distance = 1
		}
	}

	t.GetActiveBuffer().setPosition(uint16(distance-1), t.GetActiveBuffer().CursorLine())
	return true
}

func parseCursorPosition(params []string) (x, y int) {
	x, y = 1, 1
	if len(params) >= 1 {
		var err error
		if params[0] != "" {
			y, err = strconv.Atoi(string(params[0]))
			if err != nil || y < 1 {
				y = 1
			}
		}
	}
	if len(params) >= 2 {
		if params[1] != "" {
			var err error
			x, err = strconv.Atoi(string(params[1]))
			if err != nil || x < 1 {
				x = 1
			}
		}
	}
	return x, y
}

// CSI f
// Horizontal and Vertical Position [row;column] (default = [1,1]) (HVP)
// AND
// CSI H
// Cursor Position [row;column] (default = [1,1]) (CUP)
func (t *Terminal) csiCursorPositionHandler(params []string) (renderRequired bool) {
	x, y := parseCursorPosition(params)
	t.GetActiveBuffer().setPosition(uint16(x-1), uint16(y-1))
	return true
}

// CSI S
// Scroll up Ps lines (default = 1) (SU), VT420, ECMA-48
func (t *Terminal) csiScrollUpHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 1 {
		return false
	}
	if len(params) == 1 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}
	t.GetActiveBuffer().areaScrollUp(uint16(distance))
	return true
}

// CSI @
// Insert Ps (Blank) Character(s) (default = 1) (ICH)
func (t *Terminal) csiInsertBlankCharactersHandler(params []string) (renderRequired bool) {
	count := 1
	if len(params) > 1 {
		return false
	}
	if len(params) == 1 {
		var err error
		count, err = strconv.Atoi(params[0])
		if err != nil || count < 1 {
			count = 1
		}
	}

	t.GetActiveBuffer().insertBlankCharacters(count)
	return true
}

// CSI L
// Insert Ps Line(s) (default = 1) (IL)
func (t *Terminal) csiInsertLinesHandler(params []string) (renderRequired bool) {
	count := 1
	if len(params) > 1 {
		return false
	}
	if len(params) == 1 {
		var err error
		count, err = strconv.Atoi(params[0])
		if err != nil || count < 1 {
			count = 1
		}
	}

	t.GetActiveBuffer().insertLines(count)
	return true
}

// CSI M
// Delete Ps Line(s) (default = 1) (DL)
func (t *Terminal) csiDeleteLinesHandler(params []string) (renderRequired bool) {
	count := 1
	if len(params) > 1 {
		return false
	}
	if len(params) == 1 {
		var err error
		count, err = strconv.Atoi(params[0])
		if err != nil || count < 1 {
			count = 1
		}
	}

	t.GetActiveBuffer().deleteLines(count)
	return true
}

// CSI T
// Scroll down Ps lines (default = 1) (SD), VT420
func (t *Terminal) csiScrollDownHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 1 {
		return false
	}
	if len(params) == 1 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}
	t.GetActiveBuffer().areaScrollDown(uint16(distance))
	return true
}

// CSI r
// Set Scrolling Region [top;bottom] (default = full size of window) (DECSTBM), VT100
func (t *Terminal) csiSetMarginsHandler(params []string) (renderRequired bool) {
	top := 1
	bottom := int(t.GetActiveBuffer().ViewHeight())

	if len(params) > 2 {
		return false
	}

	if len(params) > 0 {
		var err error
		top, err = strconv.Atoi(params[0])
		if err != nil || top < 1 {
			top = 1
		}

		if len(params) > 1 {
			var err error
			bottom, err = strconv.Atoi(params[1])
			if err != nil || bottom > int(t.GetActiveBuffer().ViewHeight()) || bottom < 1 {
				bottom = int(t.GetActiveBuffer().ViewHeight())
			}
		}
	}
	top--
	bottom--

	t.activeBuffer.setVerticalMargins(uint(top), uint(bottom))
	t.GetActiveBuffer().setPosition(0, 0)
	return true
}

// CSI X
// Erase Ps Character(s) (default = 1) (ECH)
func (t *Terminal) csiEraseCharactersHandler(params []string) (renderRequired bool) {
	count := 1
	if len(params) > 0 {
		var err error
		count, err = strconv.Atoi(params[0])
		if err != nil || count < 1 {
			count = 1
		}
	}

	t.GetActiveBuffer().eraseCharacters(count)
	return true
}

// CSI l
// Reset Mode (RM)
func (t *Terminal) csiResetModeHandler(params []string) (renderRequired bool) {
	return t.csiSetModes(params, false)
}

// CSI h
// Set Mode (SM)
func (t *Terminal) csiSetModeHandler(params []string) (renderRequired bool) {
	return t.csiSetModes(params, true)
}

func (t *Terminal) csiSetModes(modes []string, enabled bool) bool {
	if len(modes) == 0 {
		return false
	}
	if len(modes) == 1 {
		return t.csiSetMode(modes[0], enabled)
	}
	// should we propagate DEC prefix?
	const decPrefix = '?'
	isDec := len(modes[0]) > 0 && modes[0][0] == decPrefix

	var render bool

	// iterate through params, propagating DEC prefix to subsequent elements
	for i, v := range modes {
		updatedMode := v
		if i > 0 && isDec {
			updatedMode = string(decPrefix) + v
		}
		render = t.csiSetMode(updatedMode, enabled) || render
	}

	return render
}

func parseModes(mode string) []string {
	var output []string

	if mode == "" {
		return nil
	}
	var prefix string
	if mode[0] == '?' {
		prefix = "?"
		mode = mode[1:]
	}

	for len(mode) > 4 {
		output = append(output, prefix+mode[:4])
		mode = mode[4:]
	}

	output = append(output, prefix+mode)
	return output
}

func (t *Terminal) csiSetMode(modes string, enabled bool) bool {
	for _, modeStr := range parseModes(modes) {
		switch modeStr {
		case "4":
			t.activeBuffer.modes.ReplaceMode = !enabled
		case "20":
			t.activeBuffer.modes.LineFeedMode = false
		case "?1":
			t.activeBuffer.modes.ApplicationCursorKeys = enabled
		case "?3":
			// TODO decide if supporting 132 column mode
		case "?5": // DECSCNM
			t.activeBuffer.modes.ScreenMode = enabled
		case "?6":
			// DECOM
			t.activeBuffer.modes.OriginMode = enabled
		case "?7":
			// auto-wrap mode
			// DECAWM
			t.activeBuffer.modes.AutoWrap = enabled
		case "?9":
			if enabled {
				t.mouseMode = (MouseModeX10)
			} else {
				t.mouseMode = (MouseModeNone)
			}
		case "?12", "?13":
			t.activeBuffer.modes.BlinkingCursor = enabled
		case "?25":
			t.activeBuffer.modes.ShowCursor = enabled
		case "?47", "?1047":
			if enabled {
				t.useAltBuffer()
			} else {
				t.useMainBuffer()
			}
		case "?1000": // ?10061000 seen from htop
			// enable mouse tracking
			// 1000 refers to ext mode for extended mouse click area - otherwise only x <= 255-31
			if enabled {
				t.mouseMode = (MouseModeVT200)
			} else {
				t.mouseMode = (MouseModeNone)
			}
		case "?1002":
			if enabled {
				t.mouseMode = (MouseModeButtonEvent)
			} else {
				t.mouseMode = (MouseModeNone)
			}
		case "?1003":
			if enabled {
				t.mouseMode = MouseModeAnyEvent
			} else {
				t.mouseMode = MouseModeNone
			}
		case "?1005":
			if enabled {
				t.mouseExtMode = MouseExtUTF
			} else {
				t.mouseExtMode = MouseExtNone
			}

		case "?1006":
			if enabled {
				t.mouseExtMode = MouseExtSGR
			} else {
				t.mouseExtMode = (MouseExtNone)
			}
		case "?1015":
			if enabled {
				t.mouseExtMode = (MouseExtURXVT)
			} else {
				t.mouseExtMode = (MouseExtNone)
			}
		case "?1048":
			if enabled {
				t.GetActiveBuffer().saveCursor()
			} else {
				t.GetActiveBuffer().restoreCursor()
			}
		case "?1049":
			if enabled {
				t.useAltBuffer()
			} else {
				t.useMainBuffer()
			}
		case "?2004":
			t.activeBuffer.modes.BracketedPasteMode = enabled
		case "?80":
			// t.activeBuffer.modes.SixelScrolling = enabled
		default:
			// log.Printf("Unsupported CSI mode %s = %t", modeStr, enabled)
		}
	}
	return false
}

// CSI d
// Line Position Absolute  [row] (default = [1,column]) (VPA)
func (t *Terminal) csiLinePositionAbsoluteHandler(params []string) (renderRequired bool) {
	row := 1
	if len(params) > 0 {
		var err error
		row, err = strconv.Atoi(params[0])
		if err != nil || row < 1 {
			row = 1
		}
	}

	t.GetActiveBuffer().setPosition(t.GetActiveBuffer().CursorColumn(), uint16(row-1))

	return true
}

// CSI P
// Delete Ps Character(s) (default = 1) (DCH)
func (t *Terminal) csiDeleteHandler(params []string) (renderRequired bool) {
	n := 1
	if len(params) >= 1 {
		var err error
		n, err = strconv.Atoi(params[0])
		if err != nil || n < 1 {
			n = 1
		}
	}

	t.GetActiveBuffer().deleteChars(n)
	return true
}

// CSI g
// tab clear (TBC)
func (t *Terminal) csiTabClearHandler(params []string) (renderRequired bool) {
	n := "0"
	if len(params) > 0 {
		n = params[0]
	}
	switch n {
	case "0", "":
		t.activeBuffer.tabClearAtCursor()
	case "3":
		t.activeBuffer.tabReset()
	default:
		return false
	}

	return true
}

// CSI J
// Erase in Display (ED), VT100
func (t *Terminal) csiEraseInDisplayHandler(params []string) (renderRequired bool) {
	n := "0"
	if len(params) > 0 {
		n = params[0]
	}

	switch n {
	case "0", "":
		t.GetActiveBuffer().eraseDisplayFromCursor()
	case "1":
		t.GetActiveBuffer().eraseDisplayToCursor()
	case "2", "3":
		t.GetActiveBuffer().eraseDisplay()
	default:
		return false
	}

	return true
}

// CSI K
// Erase in Line (EL), VT100
func (t *Terminal) csiEraseInLineHandler(params []string) (renderRequired bool) {
	n := "0"
	if len(params) > 0 {
		n = params[0]
	}

	switch n {
	case "0", "": // erase adter cursor
		t.GetActiveBuffer().eraseLineFromCursor()
	case "1": // erase to cursor inclusive
		t.GetActiveBuffer().eraseLineToCursor()
	case "2": // erase entire
		t.GetActiveBuffer().eraseLine()
	default:
		return false
	}
	return true
}

// CSI m
// Character Attributes (SGR)
func (t *Terminal) sgrSequenceHandler(params []string) bool {
	if len(params) == 0 {
		params = []string{"0"}
	}

	for i, p := range params {
		attr := t.GetActiveBuffer().getCursorAttr()
		switch p {
		case "00", "0", "":
			*attr = tcell.Style{}
		case "1", "01":
			*attr = attr.Bold(true)
		case "2", "02":
			*attr = attr.Dim(true)
		case "3", "03":
			*attr = attr.Italic(true)
		case "4", "04":
			*attr = attr.Underline(true)
		case "5", "05":
			*attr = attr.Blink(true)
		case "7", "07":
			*attr = attr.Reverse(true)
		case "8", "08":
			// *attr = attr.Hidden(true)
		case "9", "09":
			*attr = attr.StrikeThrough(true)
		case "21":
			*attr = attr.Bold(false)
		case "22":
			*attr = attr.Dim(false)
		case "23":
			*attr = attr.Italic(false)
		case "24":
			*attr = attr.Underline(false)
		case "25":
			*attr = attr.Blink(false)
		case "27":
			*attr = attr.Reverse(false)
		case "28":
			//*attr = attr.Hidden( false)
		case "29":
			*attr = attr.StrikeThrough(false)
		case "38": // set foreground
			fg, err := t.theme.ColourFromAnsi(params[i+1:])
			if err != nil {
				*attr = attr.Foreground(t.theme.DefaultForeground())
			} else {
				*attr = attr.Foreground(fg)
			}
			return false
		case "48": // set background
			bg, err := t.theme.ColourFromAnsi(params[i+1:])
			if err != nil {
				*attr = attr.Background(t.theme.DefaultBackground())
			} else {
				*attr = attr.Background(bg)
			}
			return false
		case "39":
			*attr = attr.Foreground(t.theme.DefaultForeground())
		case "49":
			*attr = attr.Background(t.theme.DefaultBackground())
		default:
			bi, err := strconv.Atoi(p)
			if err != nil {
				return false
			}
			i := byte(bi)
			switch true {
			case i >= 30 && i <= 37, i >= 90 && i <= 97:
				*attr = attr.Foreground(t.theme.ColourFrom4Bit(i))
			case i >= 40 && i <= 47, i >= 100 && i <= 107:
				*attr = attr.Background(t.theme.ColourFrom4Bit(i))
			}

		}
	}

	x := t.GetActiveBuffer().CursorColumn()
	y := t.GetActiveBuffer().CursorLine()
	if cell := t.GetActiveBuffer().GetCell(x, y); cell != nil {
		cell.setStyle(t.GetActiveBuffer().cursorAttr)
	}

	return false
}

func (t *Terminal) csiSoftResetHandler(params []string) bool {
	t.reset()
	return true
}

func (t *Terminal) csiCursorSelection(params []string) (renderRequired bool) {
	if len(params) == 0 {
		return false
	}
	i, err := strconv.Atoi(params[0])
	if err != nil {
		return false
	}
	t.GetActiveBuffer().SetCursorShape(CursorShape(i))
	return true
}
