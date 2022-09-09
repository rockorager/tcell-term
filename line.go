package tcellterm

import "strings"

type line struct {
	wrapped bool // whether line was wrapped onto from the previous one
	cells   []cell
}

func newLine() line {
	return line{
		wrapped: false,
		cells:   []cell{},
	}
}

func (l *line) len() uint16 {
	return uint16(len(l.cells))
}

func (l *line) string() string {
	runes := []rune{}
	for _, cell := range l.cells {
		runes = append(runes, cell.r.rune)
	}
	return strings.TrimRight(string(runes), "\x00")
}

func (l *line) append(cells ...cell) {
	l.cells = append(l.cells, cells...)
}

func (l *line) setDirty(d bool) {
	for _, cell := range l.cells {
		cell.setDirty(d)
	}
}

func (l *line) shrink(width uint16) {
	if l.len() <= width {
		return
	}
	remove := l.len() - width
	var cells []cell
	for _, cell := range l.cells {
		if cell.r.rune == 0 && remove > 0 {
			remove--
		} else {
			cell.setDirty(true)
			cells = append(cells, cell)
		}
	}
	l.cells = cells
}

func (l *line) wrap(width uint16) []line {
	var output []line
	var current line

	current.wrapped = l.wrapped

	for _, cell := range l.cells {
		cell.setDirty(true)
		if len(current.cells) == int(width) {
			output = append(output, current)
			current = newLine()
			current.wrapped = true
		}
		current.cells = append(current.cells, cell)
	}

	return append(output, current)
}
