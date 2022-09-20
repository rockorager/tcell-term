package tcellterm

func (b *buffer) shrink(width int) {
	var replace []line

	prevCursor := int(b.cursorPosition.Line)

	for i, line := range b.lines {

		line.shrink(width)

		// this line fits within the new width restriction, keep it as is and continue
		if line.len() <= width {
			replace = append(replace, line)
			continue
		}

		wrappedLines := line.wrap(width)

		if prevCursor >= i {
			b.cursorPosition.Line += len(wrappedLines) - 1
		}

		replace = append(replace, wrappedLines...)
	}

	b.cursorPosition.Col = b.cursorPosition.Col % width

	b.lines = replace
}

func (b *buffer) grow(width int) {
	var replace []line
	var current line

	prevCursor := int(b.cursorPosition.Line)

	for i, line := range b.lines {

		if !line.wrapped {
			if i > 0 {
				replace = append(replace, current)
			}
			current = newLine()
		}

		if i == prevCursor {
			b.cursorPosition.Line -= (i - len(replace))
		}

		for _, cell := range line.cells {
			if len(current.cells) == int(width) {
				replace = append(replace, current)
				current = newLine()
				current.wrapped = true
			}
			current.cells = append(current.cells, cell)
		}

	}

	replace = append(replace, current)

	b.lines = replace
}

func (b *buffer) resizeView(width int, height int) {
	if b.viewHeight == 0 {
		b.viewWidth = width
		b.viewHeight = height
		return
	}

	// scroll to bottom
	b.scrollLinesFromBottom = 0

	if width < b.viewWidth { // wrap lines if we're shrinking
		b.shrink(width)
		b.grow(width)
	} else if width > b.viewWidth { // unwrap lines if we're growing
		b.grow(width)
	}

	b.viewWidth = width
	b.viewHeight = height

	b.resetVerticalMargins(b.viewHeight)
}
