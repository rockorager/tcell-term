package tcellterm

func (t *Terminal) handleANSI(readChan chan measuredRune) (renderRequired bool) {
	// if the byte is an escape character, read the next byte to determine which one
	r := <-readChan

	switch r.rune {
	case '[':
		return t.handleCSI(readChan)
	case ']':
		return t.handleOSC(readChan)
	case '(':
		return t.handleSCS0(readChan) // select character set into G0
	case ')':
		return t.handleSCS1(readChan) // select character set into G1
	case '*':
		return swallowHandler(1)(readChan) // character set
	case '+':
		return swallowHandler(1)(readChan) // character set
	case '>':
		return swallowHandler(0)(readChan) // numeric char selection
	case '=':
		return swallowHandler(0)(readChan) // alt char selection
	case '7':
		t.getActiveBuffer().saveCursor()
	case '8':
		t.getActiveBuffer().restoreCursor()
	case 'D':
		t.getActiveBuffer().index()
	case 'E':
		t.getActiveBuffer().newLineEx(true)
	case 'H':
		t.getActiveBuffer().tabSetAtCursor()
	case 'M':
		t.getActiveBuffer().reverseIndex()
	case 'P': // Device Control (DCS)
		// TODO Add in device control handling. Or just allow sixel
		// handling if the sequence is right
		// t.handleSixel(readChan)
	case 'c':
		t.getActiveBuffer().clear()
	case '#':
		return t.handleScreenState(readChan)
	case '^':
		return t.handlePrivacyMessage(readChan)
	default:
		// log.Printf("UNKNOWN ESCAPE SEQUENCE: 0x%X", r.Rune)
		return false
	}

	return true
}

func swallowHandler(size int) func(pty chan measuredRune) bool {
	return func(pty chan measuredRune) bool {
		for i := 0; i < size; i++ {
			<-pty
		}
		return false
	}
}

func (t *Terminal) handleScreenState(readChan chan measuredRune) bool {
	b := <-readChan
	switch b.rune {
	case '8': // DECALN -- Screen Alignment Pattern

		// hide cursor?
		buffer := t.getActiveBuffer()
		buffer.resetVerticalMargins(buffer.viewHeight)
		buffer.SetScrollOffset(0)

		// Fill the whole screen with E's
		count := buffer.ViewHeight() * buffer.ViewWidth()
		for count > 0 {
			buffer.write(measuredRune{rune: 'E', width: 1})
			count--
			if count > 0 && !buffer.modes.AutoWrap && count%buffer.ViewWidth() == 0 {
				buffer.index()
				buffer.carriageReturn()
			}
		}
		// restore cursor
		buffer.setPosition(0, 0)
	default:
		return false
	}
	return true
}

func (t *Terminal) handlePrivacyMessage(readChan chan measuredRune) bool {
	isEscaped := false
	for {
		b := <-readChan
		if b.rune == 0x18 /*CAN*/ || b.rune == 0x1a /*SUB*/ || (b.rune == 0x5c /*backslash*/ && isEscaped) {
			break
		}
		if isEscaped {
			isEscaped = false
		} else if b.rune == 0x1b {
			isEscaped = true
			continue
		}
	}
	return false
}
