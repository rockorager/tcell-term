package termutil

import (
	"github.com/gdamore/tcell/v2"
)

type Cell struct {
	r    MeasuredRune
	attr tcell.Style
}

func (cell *Cell) Rune() MeasuredRune {
	return cell.r
}

func (cell *Cell) Style() tcell.Style {
	return cell.attr
}

/*
func (cell *Cell) Fg() tcell.Color {
	if cell.attr.inverse {
		return cell.attr.bgColour
	}
	return cell.attr.fgColour
}

func (cell *Cell) Bold() bool {
	return cell.attr.bold
}

func (cell *Cell) Dim() bool {
	return cell.attr.dim
}

func (cell *Cell) Italic() bool {
	return cell.attr.italic
}

func (cell *Cell) Underline() bool {
	return cell.attr.underline
}

func (cell *Cell) Strikethrough() bool {
	return cell.attr.strikethrough
}

func (cell *Cell) Bg() tcell.Color {
	if cell.attr.inverse {
		return cell.attr.fgColour
	}
	return cell.attr.bgColour
}
*/

func (cell *Cell) erase(bgColour tcell.Color) {
	cell.setRune(MeasuredRune{Rune: 0})
	cell.attr = cell.attr.Background(bgColour)
}

func (cell *Cell) setRune(r MeasuredRune) {
	cell.r = r
}
