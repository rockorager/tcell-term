package tcellterm

import (
	"fmt"

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
