package termutil

import (
	"github.com/gdamore/tcell/v2"
)

type Cell struct {
	r     MeasuredRune
	attr  tcell.Style
	dirty bool
}

func (cell *Cell) Rune() MeasuredRune {
	return cell.r
}

func (cell *Cell) Style() tcell.Style {
	return cell.attr
}

func (cell *Cell) Dirty() bool {
	return cell.dirty
}

func (cell *Cell) SetDirty(d bool) {
	cell.dirty = d
}

func (cell *Cell) erase(bgColour tcell.Color) {
	cell.setRune(MeasuredRune{Rune: 0})
	cell.attr = cell.attr.Background(bgColour)
	cell.SetDirty(true)
}

func (cell *Cell) setRune(r MeasuredRune) {
	cell.r = r
	cell.SetDirty(true)
}

func (cell *Cell) setStyle(s tcell.Style) {
	cell.attr = s
	cell.SetDirty(true)
}
