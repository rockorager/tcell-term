package tcellterm

func (b *buffer) ClearSelection() {
	b.selectionMu.Lock()
	defer b.selectionMu.Unlock()
	b.selectionStart = nil
	b.selectionEnd = nil
}

func (b *buffer) GetBoundedTextAtPosition(pos position) (start position, end position, text string, textIndex int, found bool) {
	return b.FindWordAt(pos, func(r rune) bool {
		return r > 0 && r < 256
	})
}

// if the selection is invalid - e.g. lines are selected that no longer exist in the buffer
func (b *buffer) fixSelection() bool {
	b.selectionMu.Lock()
	defer b.selectionMu.Unlock()

	if b.selectionStart == nil || b.selectionEnd == nil {
		return false
	}

	if b.selectionStart.Line >= len(b.lines) {
		b.selectionStart.Line = len(b.lines) - 1
	}

	if b.selectionEnd.Line >= len(b.lines) {
		b.selectionEnd.Line = len(b.lines) - 1
	}

	if b.selectionStart.Col >= len(b.lines[b.selectionStart.Line].cells) {
		b.selectionStart.Col = 0
		if b.selectionStart.Line < len(b.lines)-1 {
			b.selectionStart.Line++
		}
	}

	if b.selectionEnd.Col >= len(b.lines[b.selectionEnd.Line].cells) {
		b.selectionEnd.Col = len(b.lines[b.selectionEnd.Line].cells) - 1
	}

	return true
}

func (b *buffer) ExtendSelectionToEntireLines() {
	if !b.fixSelection() {
		return
	}

	b.selectionMu.Lock()
	defer b.selectionMu.Unlock()

	b.selectionStart.Col = 0
	b.selectionEnd.Col = len(b.lines[b.selectionEnd.Line].cells) - 1
}

type runeMatcher func(r rune) bool

func (b *buffer) SelectWordAt(pos position, rm runeMatcher) {
	start, end, _, _, found := b.FindWordAt(pos, rm)
	if !found {
		return
	}
	b.setRawSelectionStart(start)
	b.setRawSelectionEnd(end)
}

// takes raw coords
func (b *buffer) Highlight(start position, end position, ann *annotation) {
	b.highlightStart = &start
	b.highlightEnd = &end
	b.highlightAnnotation = ann
}

func (b *buffer) ClearHighlight() {
	b.highlightStart = nil
	b.highlightEnd = nil
}

// returns raw lines
func (b *buffer) FindWordAt(pos position, rm runeMatcher) (start position, end position, text string, textIndex int, found bool) {
	line := b.convertViewLineToRawLine(pos.Line)
	col := pos.Col

	if line >= len(b.lines) {
		return
	}
	if col >= len(b.lines[line].cells) {
		return
	}

	if !rm(b.lines[line].cells[col].r.rune) {
		return
	}

	found = true

	start = position{
		Line: line,
		Col:  col,
	}
	end = position{
		Line: line,
		Col:  col,
	}

	var startCol int
BACK:
	for y := line; y >= 0; y-- {
		if y == line {
			startCol = col
		} else {
			if len(b.lines[y].cells) < int(b.viewWidth) {
				break
			}
			startCol = len(b.lines[y].cells) - 1
		}
		for x := startCol; x >= 0; x-- {
			if rm(b.lines[y].cells[x].r.rune) {
				start = position{
					Line: y,
					Col:  x,
				}
				text = string(b.lines[y].cells[x].r.rune) + text
			} else {
				break BACK
			}
		}

	}
	textIndex = len([]rune(text)) - 1
FORWARD:
	for y := line; y < len(b.lines); y++ {
		if y == line {
			startCol = col + 1
		} else {
			startCol = 0
		}
		for x := startCol; x < len(b.lines[y].cells); x++ {
			if rm(b.lines[y].cells[x].r.rune) {
				end = position{
					Line: y,
					Col:  x,
				}
				text = text + string(b.lines[y].cells[x].r.rune)
			} else {
				break FORWARD
			}
		}
		if len(b.lines[y].cells) < int(b.viewWidth) {
			break
		}
	}

	return
}

func (b *buffer) SetSelectionStart(pos position) {
	b.selectionMu.Lock()
	defer b.selectionMu.Unlock()
	b.selectionStart = &position{
		Col:  pos.Col,
		Line: b.convertViewLineToRawLine(pos.Line),
	}
}

func (b *buffer) setRawSelectionStart(pos position) {
	b.selectionMu.Lock()
	defer b.selectionMu.Unlock()
	b.selectionStart = &pos
}

func (b *buffer) SetSelectionEnd(pos position) {
	b.selectionMu.Lock()
	defer b.selectionMu.Unlock()
	b.selectionEnd = &position{
		Col:  pos.Col,
		Line: b.convertViewLineToRawLine(pos.Line),
	}
}

func (b *buffer) setRawSelectionEnd(pos position) {
	b.selectionMu.Lock()
	defer b.selectionMu.Unlock()
	b.selectionEnd = &pos
}

func (b *buffer) GetSelection() (string, *selection) {
	if !b.fixSelection() {
		return "", nil
	}

	b.selectionMu.Lock()
	defer b.selectionMu.Unlock()

	start := *b.selectionStart
	end := *b.selectionEnd

	if end.Line < start.Line || (end.Line == start.Line && end.Col < start.Col) {
		swap := end
		end = start
		start = swap
	}

	var text string
	for y := start.Line; y <= end.Line; y++ {
		if y >= len(b.lines) {
			break
		}
		line := b.lines[y]
		startX := 0
		endX := len(line.cells) - 1
		if y == start.Line {
			startX = int(start.Col)
		}
		if y == end.Line {
			endX = int(end.Col)
		}
		if y > start.Line {
			text += "\n"
		}
		for x := startX; x <= endX; x++ {
			if x >= len(line.cells) {
				break
			}
			mr := line.cells[x].rune()
			if mr.width == 0 {
				continue
			}
			x += mr.width - 1
			text += string(mr.rune)
		}
	}

	viewSelection := selection{
		Start: start,
		End:   end,
	}

	viewSelection.Start.Line = b.convertRawLineToViewLine(viewSelection.Start.Line)
	viewSelection.End.Line = b.convertRawLineToViewLine(viewSelection.End.Line)
	return text, &viewSelection
}

func (b *buffer) InSelection(pos position) bool {
	if !b.fixSelection() {
		return false
	}
	b.selectionMu.Lock()
	defer b.selectionMu.Unlock()

	start := *b.selectionStart
	end := *b.selectionEnd

	if end.Line < start.Line || (end.Line == start.Line && end.Col < start.Col) {
		swap := end
		end = start
		start = swap
	}

	rY := b.convertViewLineToRawLine(pos.Line)
	if rY < start.Line {
		return false
	}
	if rY > end.Line {
		return false
	}
	if rY == start.Line {
		if pos.Col < start.Col {
			return false
		}
	}
	if rY == end.Line {
		if pos.Col > end.Col {
			return false
		}
	}

	return true
}

func (b *buffer) GetHighlightAnnotation() *annotation {
	return b.highlightAnnotation
}

func (b *buffer) GetViewHighlight() (start position, end position, exists bool) {
	if b.highlightStart == nil || b.highlightEnd == nil {
		return
	}

	if b.highlightStart.Line >= len(b.lines) {
		return
	}

	if b.highlightEnd.Line >= len(b.lines) {
		return
	}

	if b.highlightStart.Col >= len(b.lines[b.highlightStart.Line].cells) {
		return
	}

	if b.highlightEnd.Col >= len(b.lines[b.highlightEnd.Line].cells) {
		return
	}

	start = *b.highlightStart
	end = *b.highlightEnd

	if end.Line < start.Line || (end.Line == start.Line && end.Col < start.Col) {
		swap := end
		end = start
		start = swap
	}

	start.Line = b.convertRawLineToViewLine(start.Line)
	end.Line = b.convertRawLineToViewLine(end.Line)

	return start, end, true
}
