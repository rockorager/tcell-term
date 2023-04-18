package tcellterm

func (vt *VT) c0(r rune) {
	switch r {
	case 0x07:
		vt.postEvent(EventBell{
			EventTerminal: newEventTerminal(vt),
		})
	case 0x08:
		vt.bs()
	case 0x09:
		vt.ht()
	case 0x0A:
		vt.lf()
	case 0x0B:
		vt.vt()
	case 0x0C:
		vt.ff()
	case 0x0D:
		vt.cr()
	case 0x0E:
		vt.charset = vt.g1
	case 0x0F:
		vt.charset = vt.g0
	}
}

// Backspace 0x08
func (vt *VT) bs() {
	vt.lastCol = false
	if vt.cursor.col == vt.margin.left {
		return
	}
	vt.cursor.col -= 1
}

// Horizontal tab 0x09
func (vt *VT) ht() {
	// TODO
}

// Linefeed 0x10
func (vt *VT) lf() {
	vt.ind()

	if vt.mode&lnm != lnm {
		return
	}
	vt.cursor.col = vt.margin.left
}

// Vertical tabulation 0x11
func (vt *VT) vt() {
	vt.lf()
}

// Form feed 0x12
func (vt *VT) ff() {
	vt.lf()
}

// Carriage return 0x13
func (vt *VT) cr() {
	vt.lastCol = false
	vt.cursor.col = vt.margin.left
}
