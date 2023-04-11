package tcellterm

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
)

var (
	linux_key_map = map[tcell.Key]string{
		tcell.KeyEnter:      "\r",
		tcell.KeyBackspace:  "\u007f",
		tcell.KeyBackspace2: "\x7f",
		tcell.KeyTab:        "\t",
		tcell.KeyEscape:     "\x1b",
		tcell.KeyDown:       "\x1bOB",
		tcell.KeyUp:         "\x1bOA",
		tcell.KeyRight:      "\x1bOC",
		tcell.KeyLeft:       "\x1bOD",
		tcell.KeyHome:       "\x1bOH",
		tcell.KeyEnd:        "\x1bOF",
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
		tcell.KeyF11:        "\x1b[23~",
		tcell.KeyF12:        "\x1b[24~",
		/*
			"bracketed_paste_mode_start": "\x1b[200~",
			"bracketed_paste_mode_end":   "\x1b[201~",
		*/
	}

	linux_ctrl_key_map = map[tcell.Key]string{
		tcell.KeyUp:    "\x1b[1;5A",
		tcell.KeyDown:  "\x1b[1;5B",
		tcell.KeyRight: "\x1b[1;5C",
		tcell.KeyLeft:  "\x1b[1;5D",
	}

	linux_ctrl_rune_map = map[rune]string{
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

	linux_alt_key_map = map[tcell.Key]string{
		tcell.KeyUp:    "\x1b[1;3A",
		tcell.KeyDown:  "\x1b[1;3B",
		tcell.KeyRight: "\x1b[1;3C",
		tcell.KeyLeft:  "\x1b[1;3D",
	}

	linux_ctrl_alt_key_map = map[tcell.Key]string{
		tcell.KeyUp:    "\x1b[1;7A",
		tcell.KeyDown:  "\x1b[1;7B",
		tcell.KeyRight: "\x1b[1;7C",
		tcell.KeyLeft:  "\x1b[1;7D",
	}
)

func getCtrlCombinationKeyCode(ke *tcell.EventKey) string {
	if keycode, ok := linux_ctrl_key_map[ke.Key()]; ok {
		return keycode
	}
	if keycode, ok := linux_ctrl_rune_map[ke.Rune()]; ok {
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
	if ke.Modifiers()&tcell.ModCtrl != 0 {
		if keycode, ok := linux_ctrl_alt_key_map[ke.Key()]; ok {
			return keycode
		}
	}
	if keycode, ok := linux_alt_key_map[ke.Key()]; ok {
		return keycode
	}
	if ke.Key() != tcell.KeyRune {
		return fmt.Sprintf("\x1b%c", ke.Key())
	}
	code := getKeyCode(ke)
	return "\x1b" + code
}

func getKeyCode(ke *tcell.EventKey) string {
	if keycode, ok := linux_key_map[ke.Key()]; ok {
		return keycode
	}
	return string(ke.Rune())
}

func keyCode(ev *tcell.EventKey) string {
	key := strings.Builder{}
	switch ev.Modifiers() {
	case tcell.ModNone:
		switch ev.Key() {
		case tcell.KeyRune:
			key.WriteRune(ev.Rune())
		default:
			if str, ok := keyCodes[ev.Key()]; ok {
				key.WriteString(str)
			} else {
				key.WriteRune(rune(ev.Key()))
			}
		}
	case tcell.ModShift:
		switch ev.Key() {
		case tcell.KeyUp:
			key.WriteString(info.KeyShfUp)
		case tcell.KeyDown:
			key.WriteString(info.KeyShfDown)
		case tcell.KeyRight:
			key.WriteString(info.KeyShfRight)
		case tcell.KeyLeft:
			key.WriteString(info.KeyShfLeft)
		case tcell.KeyHome:
			key.WriteString(info.KeyShfHome)
		case tcell.KeyEnd:
			key.WriteString(info.KeyShfEnd)
		case tcell.KeyInsert:
			key.WriteString(info.KeyShfInsert)
		case tcell.KeyDelete:
			key.WriteString(info.KeyShfDelete)
		case tcell.KeyPgUp:
			key.WriteString(info.KeyShfPgUp)
		case tcell.KeyPgDn:
			key.WriteString(info.KeyShfPgDn)
		}
	case tcell.ModAlt:
		switch ev.Key() {
		case tcell.KeyRune:
			key.WriteString("\x1b[")
			key.WriteRune(ev.Rune())
		case tcell.KeyUp:
			key.WriteString(info.KeyAltUp)
		case tcell.KeyDown:
			key.WriteString(info.KeyAltDown)
		case tcell.KeyRight:
			key.WriteString(info.KeyAltRight)
		case tcell.KeyLeft:
			key.WriteString(info.KeyAltLeft)
		case tcell.KeyHome:
			key.WriteString(info.KeyAltHome)
		case tcell.KeyEnd:
			key.WriteString(info.KeyAltEnd)
		}
	case tcell.ModCtrl:
		switch ev.Key() {
		case tcell.KeyUp:
			key.WriteString(info.KeyCtrlUp)
		case tcell.KeyDown:
			key.WriteString(info.KeyCtrlDown)
		case tcell.KeyRight:
			key.WriteString(info.KeyCtrlRight)
		case tcell.KeyLeft:
			key.WriteString(info.KeyCtrlLeft)
		case tcell.KeyHome:
			key.WriteString(info.KeyCtrlHome)
		case tcell.KeyEnd:
			key.WriteString(info.KeyCtrlEnd)
		default:
			key.WriteRune(ev.Rune())
		}
	case tcell.ModCtrl | tcell.ModShift:
		switch ev.Key() {
		case tcell.KeyUp:
			key.WriteString(info.KeyCtrlShfUp)
		case tcell.KeyDown:
			key.WriteString(info.KeyCtrlShfDown)
		case tcell.KeyRight:
			key.WriteString(info.KeyCtrlShfRight)
		case tcell.KeyLeft:
			key.WriteString(info.KeyCtrlShfLeft)
		case tcell.KeyHome:
			key.WriteString(info.KeyCtrlShfHome)
		case tcell.KeyEnd:
			key.WriteString(info.KeyCtrlShfEnd)
		}
	case tcell.ModAlt | tcell.ModShift:
		switch ev.Key() {
		case tcell.KeyUp:
			key.WriteString(info.KeyAltShfUp)
		case tcell.KeyDown:
			key.WriteString(info.KeyAltShfDown)
		case tcell.KeyRight:
			key.WriteString(info.KeyAltShfRight)
		case tcell.KeyLeft:
			key.WriteString(info.KeyAltShfLeft)
		case tcell.KeyHome:
			key.WriteString(info.KeyAltShfHome)
		case tcell.KeyEnd:
			key.WriteString(info.KeyAltShfEnd)
		}
	}
	return key.String()
}

var keyCodes = map[tcell.Key]string{
	tcell.KeyBackspace: info.KeyBackspace,
	tcell.KeyF1:        info.KeyF1,
	tcell.KeyF2:        info.KeyF2,
	tcell.KeyF3:        info.KeyF3,
	tcell.KeyF4:        info.KeyF4,
	tcell.KeyF5:        info.KeyF5,
	tcell.KeyF6:        info.KeyF6,
	tcell.KeyF7:        info.KeyF7,
	tcell.KeyF8:        info.KeyF8,
	tcell.KeyF9:        info.KeyF9,
	tcell.KeyF10:       info.KeyF10,
	tcell.KeyF11:       info.KeyF11,
	tcell.KeyF12:       info.KeyF12,
	tcell.KeyF13:       info.KeyF13,
	tcell.KeyF14:       info.KeyF14,
	tcell.KeyF15:       info.KeyF15,
	tcell.KeyF16:       info.KeyF16,
	tcell.KeyF17:       info.KeyF17,
	tcell.KeyF18:       info.KeyF18,
	tcell.KeyF19:       info.KeyF19,
	tcell.KeyF20:       info.KeyF20,
	tcell.KeyF21:       info.KeyF21,
	tcell.KeyF22:       info.KeyF22,
	tcell.KeyF23:       info.KeyF23,
	tcell.KeyF24:       info.KeyF24,
	tcell.KeyF25:       info.KeyF25,
	tcell.KeyF26:       info.KeyF26,
	tcell.KeyF27:       info.KeyF27,
	tcell.KeyF28:       info.KeyF28,
	tcell.KeyF29:       info.KeyF29,
	tcell.KeyF30:       info.KeyF30,
	tcell.KeyF31:       info.KeyF31,
	tcell.KeyF32:       info.KeyF32,
	tcell.KeyF33:       info.KeyF33,
	tcell.KeyF34:       info.KeyF34,
	tcell.KeyF35:       info.KeyF35,
	tcell.KeyF36:       info.KeyF36,
	tcell.KeyF37:       info.KeyF37,
	tcell.KeyF38:       info.KeyF38,
	tcell.KeyF39:       info.KeyF39,
	tcell.KeyF40:       info.KeyF40,
	tcell.KeyF41:       info.KeyF41,
	tcell.KeyF42:       info.KeyF42,
	tcell.KeyF43:       info.KeyF43,
	tcell.KeyF44:       info.KeyF44,
	tcell.KeyF45:       info.KeyF45,
	tcell.KeyF46:       info.KeyF46,
	tcell.KeyF47:       info.KeyF47,
	tcell.KeyF48:       info.KeyF48,
	tcell.KeyF49:       info.KeyF49,
	tcell.KeyF50:       info.KeyF50,
	tcell.KeyF51:       info.KeyF51,
	tcell.KeyF52:       info.KeyF52,
	tcell.KeyF53:       info.KeyF53,
	tcell.KeyF54:       info.KeyF54,
	tcell.KeyF55:       info.KeyF55,
	tcell.KeyF56:       info.KeyF56,
	tcell.KeyF57:       info.KeyF57,
	tcell.KeyF58:       info.KeyF58,
	tcell.KeyF59:       info.KeyF59,
	tcell.KeyF60:       info.KeyF60,
	tcell.KeyF61:       info.KeyF61,
	tcell.KeyF62:       info.KeyF62,
	tcell.KeyF63:       info.KeyF63,
	tcell.KeyF64:       info.KeyF64,
	tcell.KeyInsert:    info.KeyInsert,
	tcell.KeyDelete:    info.KeyDelete,
	tcell.KeyHome:      info.KeyHome,
	tcell.KeyEnd:       info.KeyEnd,
	tcell.KeyHelp:      info.KeyHelp,
	tcell.KeyPgUp:      info.KeyPgUp,
	tcell.KeyPgDn:      info.KeyPgDn,
	tcell.KeyUp:        info.KeyUp,
	tcell.KeyDown:      info.KeyDown,
	tcell.KeyLeft:      info.KeyLeft,
	tcell.KeyRight:     info.KeyRight,
	tcell.KeyBacktab:   info.KeyBacktab,
	tcell.KeyExit:      info.KeyExit,
	tcell.KeyClear:     info.KeyClear,
	tcell.KeyPrint:     info.KeyPrint,
	tcell.KeyCancel:    info.KeyCancel,
}
