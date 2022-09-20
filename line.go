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

func (l *line) len() int {
	return len(l.cells)
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

func (l *line) shrink(width int) {
	if l.len() <= width {
		return
	}
	remove := l.len() - width
	var cells []cell
	for _, cell := range l.cells {
		if cell.r.rune == 0 && remove > 0 {
			remove--
		} else {
			cells = append(cells, cell)
		}
	}
	l.cells = cells
}
