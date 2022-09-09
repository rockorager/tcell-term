package tcellterm

import (
	"fmt"
	"strconv"

	"github.com/gdamore/tcell/v2"
)


func defaultBackground() tcell.Color {
	_, bg, _ := tcell.StyleDefault.Decompose()
	return bg
}

func defaultForeground() tcell.Color {
	fg, _, _ := tcell.StyleDefault.Decompose()
	return fg
}

func colorFrom8Bit(n string) (tcell.Color, error) {
	index, err := strconv.Atoi(n)
	if err != nil {
		return tcell.ColorDefault, err
	}
	return tcell.PaletteColor(index), nil
}

func colorFrom24Bit(r, g, b string) (tcell.Color, error) {
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

func colorFromAnsi(ansi []string) (tcell.Color, error) {
	if len(ansi) == 0 {
		return tcell.ColorDefault, fmt.Errorf("invalid ansi colour code")
	}

	switch ansi[0] {
	case "2":
		if len(ansi) != 4 {
			return tcell.ColorDefault, fmt.Errorf("invalid 24-bit ansi colour code")
		}
		return colorFrom24Bit(ansi[1], ansi[2], ansi[3])
	case "5":
		if len(ansi) != 2 {
			return tcell.ColorDefault, fmt.Errorf("invalid 8-bit ansi colour code")
		}
		return colorFrom8Bit(ansi[1])
	default:
		return tcell.ColorDefault, fmt.Errorf("invalid ansi colour code")
	}
}
