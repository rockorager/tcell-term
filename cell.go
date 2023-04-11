package tcellterm

import "github.com/gdamore/tcell/v2"

type cell struct {
	content rune
	attrs   tcell.Style
}

func (c *cell) rune() rune {
	if c.content == rune(0) {
		return ' '
	}
	return c.content
}

// Erasing removes characters from the screen without affecting other characters
// on the screen. Erased characters are lost. The cursor position does not
// change when erasing characters or lines. Erasing a character also erases any
// character attribute of the character and applies the passed style
func (c *cell) erase(s tcell.Style) {
	c.content = ' '
	c.attrs = s
}

// selectiveErase removes the cell content, but keeps the attributes
func (c *cell) selectiveErase() {
	c.content = ' '
}
