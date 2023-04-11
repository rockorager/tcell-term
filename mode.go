package tcellterm

type mode int

const (
	// ANSI-Standardized modes
	//
	// Keyboard Action mode
	kam mode = 1 << iota
	// Insert/Replace mode
	irm
	// Send/Receive mode
	srm
	// Line feed/new line mode
	lnm

	// ANSI-Compatible DEC Private Modes
	//
	// Cursor Key mode
	decckm
	// ANSI/VT52 mode
	decanm
	// Column mode
	deccolm
	// Scroll mode
	decsclm
	// Origin mode
	decom
	// Autowrap mode
	decawm
	// Autorepeat mode
	decarm
	// Printer form feed mode
	decpff
	// Printer extent mode
	decpex
	// Text Cursor Enable mode
	dectcem
	// National replacement character sets
	decnrcm

	// xterm
	//
	// Use alternate screen
	smcup
)

func (vt *VT) sm(params []int) {
	for _, param := range params {
		switch param {
		case 2:
			vt.mode |= kam
		case 4:
			vt.mode |= irm
		case 12:
			vt.mode |= srm
		case 20:
			vt.mode |= lnm
		}
	}
}

func (vt *VT) rm(params []int) {
	for _, param := range params {
		switch param {
		case 2:
			vt.mode &^= kam
		case 4:
			vt.mode &^= irm
		case 12:
			vt.mode &^= srm
		case 20:
			vt.mode &^= lnm
		}
	}
}

func (vt *VT) decset(params []int) {
	for _, param := range params {
		switch param {
		case 1:
			vt.mode |= decckm
		case 2:
			vt.mode |= decanm
		case 3:
			vt.mode |= deccolm
		case 4:
			vt.mode |= decsclm
		case 5:
		case 6:
			vt.mode |= decom
		case 7:
			vt.mode |= decawm
		case 8:
			vt.mode |= decarm
		case 25:
			vt.mode |= dectcem
		case 1049:
			vt.decsc()
			vt.activeScreen = vt.altScreen
			vt.mode |= smcup
		}
	}
}

func (vt *VT) decrst(params []int) {
	for _, param := range params {
		switch param {
		case 1:
			vt.mode &^= decckm
		case 2:
			vt.mode &^= decanm
		case 3:
			vt.mode &^= deccolm
		case 4:
			vt.mode &^= decsclm
		case 5:
		case 6:
			vt.mode &^= decom
		case 7:
			vt.mode &^= decawm
		case 8:
			vt.mode &^= decarm
		case 25:
			vt.mode &^= dectcem
		case 1049:
			vt.decrc()
			vt.activeScreen = vt.primaryScreen
			vt.mode &^= smcup
		}
	}
}
