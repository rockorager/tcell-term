package tcellterm

import (
	"github.com/gdamore/tcell/v2"
)

type measuredRune struct {
	rune  rune
	width int
}

type cell struct {
	r     measuredRune
	attr  tcell.Style
	dirty bool
}

func (c *cell) rune() measuredRune {
	return c.r
}

func (c *cell) style() tcell.Style {
	return c.attr
}

func (c *cell) isDirty() bool {
	return c.dirty
}

func (c *cell) setDirty(d bool) {
	c.dirty = d
}

func (c *cell) erase(bgColour tcell.Color) {
	c.setRune(measuredRune{rune: 0})
	c.attr = c.attr.Background(bgColour)
	c.setDirty(true)
}

func (c *cell) setRune(r measuredRune) {
	c.r = r
	c.setDirty(true)
}

func (c *cell) setStyle(s tcell.Style) {
	c.attr = s
	c.setDirty(true)
}
