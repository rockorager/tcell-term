package tcellterm

import (
	"github.com/gdamore/tcell/v2"
)

type cursor struct {
	attrs tcell.Style
	style tcell.CursorStyle

	// position
	row row // 0-indexed
	col column // 0-indexed
}

// Returns the current cursor position: x, y. 0,0 is the top left position
func (c *cursor) position() (column, row) {
	return c.col, c.row
}

func (c *cursor) attributes() tcell.Style {
	return c.attrs
}
