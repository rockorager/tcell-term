package tcellterm

import (
	"github.com/gdamore/tcell/v2"
)

func (vt *VT) esc(esc string) {
	switch esc {
	case "7":
		vt.decsc()
	case "8":
		vt.decrc()
	case "D":
		vt.ind()
	case "E":
		vt.nel()
	case "H":
		vt.hts()
	case "M":
		vt.ri()
	case "N":
		vt.sShift = vt.charset
		vt.charset = vt.g2
	case "O":
		vt.sShift = vt.charset
		vt.charset = vt.g3
	case "=":
		// DECKPAM
	case ">":
		// DECKPNM
	case "c":
		vt.ris()
	case "(0":
		vt.g0 = decSpecialAndLineDrawing
	case ")0":
		vt.g1 = decSpecialAndLineDrawing
	case "*0":
		vt.g2 = decSpecialAndLineDrawing
	case "+0":
		vt.g3 = decSpecialAndLineDrawing
	case "(B":
		vt.g0 = ascii
	case ")B":
		vt.g1 = ascii
	case "*B":
		vt.g2 = ascii
	case "+B":
		vt.g3 = ascii
	}
}

// Index ESC-D
func (vt *VT) ind() {
	if vt.cursor.row == vt.margin.bottom {
		vt.scrollUp(1)
		return
	}
	vt.cursor.row += 1
}

// Next line ESC-E
// Moves cursor to the left margin of the next line, scrolling if necessary
func (vt *VT) nel() {
	vt.ind()
	vt.cursor.col = vt.margin.left
}

// Horizontal tab set ESC-H
func (vt *VT) hts() {
	vt.tabStop = append(vt.tabStop, vt.cursor.col)
}

// Reverse Index ESC-M
func (vt *VT) ri() {
	if vt.cursor.row == vt.margin.top {
		vt.scrollDown(1)
		return
	}
	vt.cursor.row -= 1
}

// Save Cursor DECSC ESC-7
func (vt *VT) decsc() {
	vt.savedCursor = vt.cursor
	vt.savedDECAWM = vt.mode&decawm != 0
	vt.savedDECOM = vt.mode&decom != 0
}

// Restore Cursor DECRC ESC-8
func (vt *VT) decrc() {
	vt.cursor = vt.savedCursor

	switch vt.savedDECAWM {
	case true:
		vt.mode |= decawm
	case false:
		vt.mode &^= decawm
	}

	switch vt.savedDECOM {
	case true:
		vt.mode |= decom
	case false:
		vt.mode &^= decom
	}
}

// Reset Initial State (RIS) ESC-c
func (vt *VT) ris() {
	w := vt.width()
	h := vt.height()
	vt.Resize(w, h)
	vt.cursor.attrs = tcell.StyleDefault
}