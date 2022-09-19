package tcellterm

import (
	"fmt"
)

func (t *Terminal) handleOSC(readChan chan measuredRune) (renderRequired bool) {
	params := []string{}
	param := ""

READ:
	for {
		select {
		case b := <-readChan:
			if t.isOSCTerminator(b.rune) {
				params = append(params, param)
				break READ
			}
			if b.rune == ';' {
				params = append(params, param)
				param = ""
				continue
			}
			param = fmt.Sprintf("%s%c", param, b.rune)
		default:
			return false
		}
	}

	if len(params) == 0 {
		return false
	}

	pT := params[len(params)-1]
	pS := params[:len(params)-1]

	if len(pS) == 0 {
		pS = []string{pT}
		pT = ""
	}

	switch pS[0] {
	case "0", "2", "l":
		t.setTitle(pT)
	case "8":
		attr := t.getActiveBuffer().getCursorAttr()
		*attr = attr.Url(pT)
	case "10": // get/set foreground colour
		if len(pS) > 1 {
			if pS[1] == "?" {
				t.writeToPty([]byte("\x1b]10;15"))
			}
		}
	case "11": // get/set background colour
		if len(pS) > 1 {
			if pS[1] == "?" {
				t.writeToPty([]byte("\x1b]10;0"))
			}
		}
	}
	return false
}

func (t *Terminal) isOSCTerminator(r rune) bool {
	for _, terminator := range oscTerminators {
		if terminator == r {
			return true
		}
	}
	return false
}
