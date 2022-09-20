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
