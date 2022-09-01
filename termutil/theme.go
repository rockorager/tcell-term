package termutil

import (
	"fmt"
	"strconv"

	"github.com/gdamore/tcell/v2"
)

type Colour uint8

// See https://en.wikipedia.org/wiki/ANSI_escape_code#3-bit_and_4-bit
const (
	ColourBlack Colour = iota
	ColourRed
	ColourGreen
	ColourYellow
	ColourBlue
	ColourMagenta
	ColourCyan
	ColourWhite
	ColourBrightBlack
	ColourBrightRed
	ColourBrightGreen
	ColourBrightYellow
	ColourBrightBlue
	ColourBrightMagenta
	ColourBrightCyan
	ColourBrightWhite
	ColourBackground
	ColourForeground
	ColourSelectionBackground
	ColourSelectionForeground
	ColourCursorForeground
	ColourCursorBackground
)

type Theme struct {
	colourMap map[Colour]tcell.Color
}

var map4Bit = map[uint8]Colour{
	30:  ColourBlack,
	31:  ColourRed,
	32:  ColourGreen,
	33:  ColourYellow,
	34:  ColourBlue,
	35:  ColourMagenta,
	36:  ColourCyan,
	37:  ColourWhite,
	90:  ColourBrightBlack,
	91:  ColourBrightRed,
	92:  ColourBrightGreen,
	93:  ColourBrightYellow,
	94:  ColourBrightBlue,
	95:  ColourBrightMagenta,
	96:  ColourBrightCyan,
	97:  ColourBrightWhite,
	40:  ColourBlack,
	41:  ColourRed,
	42:  ColourGreen,
	43:  ColourYellow,
	44:  ColourBlue,
	45:  ColourMagenta,
	46:  ColourCyan,
	47:  ColourWhite,
	100: ColourBrightBlack,
	101: ColourBrightRed,
	102: ColourBrightGreen,
	103: ColourBrightYellow,
	104: ColourBrightBlue,
	105: ColourBrightMagenta,
	106: ColourBrightCyan,
	107: ColourBrightWhite,
}

func (t *Theme) ColourFrom4Bit(code uint8) tcell.Color {
	colour, ok := map4Bit[code]
	if !ok {
		return tcell.ColorBlack
	}
	return tcell.PaletteColor(int(colour))
}

func (t *Theme) DefaultBackground() tcell.Color {
	_, bg, _ := tcell.StyleDefault.Decompose()
	return bg
}

func (t *Theme) DefaultForeground() tcell.Color {
	fg, _, _ := tcell.StyleDefault.Decompose()
	return fg
}

func (t *Theme) CursorBackground() tcell.Color {
	c, ok := t.colourMap[ColourCursorBackground]
	if !ok {
		return tcell.ColorWhite
	}
	return c
}

func (t *Theme) CursorForeground() tcell.Color {
	c, ok := t.colourMap[ColourCursorForeground]
	if !ok {
		return tcell.ColorBlack
	}
	return c
}

func (t *Theme) ColourFrom8Bit(n string) (tcell.Color, error) {
	index, err := strconv.Atoi(n)
	if err != nil {
		return tcell.ColorDefault, err
	}
	return tcell.PaletteColor(index), nil
}

func (t *Theme) ColourFrom24Bit(r, g, b string) (tcell.Color, error) {
	ri, err := strconv.Atoi(r)
	if err != nil {
		return tcell.ColorDefault, err
	}
	gi, err := strconv.Atoi(g)
	if err != nil {
		return tcell.ColorDefault, err
	}
	bi, err := strconv.Atoi(b)
	if err != nil {
		return tcell.ColorDefault, err
	}
	return tcell.NewRGBColor(int32(ri), int32(gi), int32(bi)), nil
}

func (t *Theme) ColourFromAnsi(ansi []string) (tcell.Color, error) {
	if len(ansi) == 0 {
		return tcell.ColorDefault, fmt.Errorf("invalid ansi colour code")
	}

	switch ansi[0] {
	case "2":
		if len(ansi) != 4 {
			return tcell.ColorDefault, fmt.Errorf("invalid 24-bit ansi colour code")
		}
		return t.ColourFrom24Bit(ansi[1], ansi[2], ansi[3])
	case "5":
		if len(ansi) != 2 {
			return tcell.ColorDefault, fmt.Errorf("invalid 8-bit ansi colour code")
		}
		return t.ColourFrom8Bit(ansi[1])
	default:
		return tcell.ColorDefault, fmt.Errorf("invalid ansi colour code")
	}
}
