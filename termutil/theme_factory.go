package termutil

import (
	"github.com/gdamore/tcell/v2"
)

type ThemeFactory struct {
	theme     *Theme
	colourMap map[Colour]tcell.Color
}

func NewThemeFactory() *ThemeFactory {
	return &ThemeFactory{
		theme: &Theme{
			colourMap: map[Colour]tcell.Color{},
		},
		colourMap: make(map[Colour]tcell.Color),
	}
}

func (t *ThemeFactory) Build() *Theme {
	for id, col := range t.colourMap {
		r, g, b := col.RGB()
		t.theme.colourMap[id] = tcell.NewRGBColor(
			int32(r/0xff),
			int32(g/0xff),
			int32(b/0xff),
		)
	}
	return t.theme
}

func (t *ThemeFactory) WithColour(key Colour, colour tcell.Color) *ThemeFactory {
	t.colourMap[key] = colour
	return t
}
