package tcellterm

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
)

func TestSGR(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected func() tcell.Style
	}{
		{
			name:  "default",
			input: []int{},
			expected: func() tcell.Style {
				return tcell.StyleDefault
			},
		},
		{
			name:  "default",
			input: []int{0},
			expected: func() tcell.Style {
				return tcell.StyleDefault
			},
		},
		{
			name:  "bold",
			input: []int{1},
			expected: func() tcell.Style {
				return tcell.StyleDefault.Bold(true)
			},
		},
		{
			name:  "underline",
			input: []int{2},
			expected: func() tcell.Style {
				return tcell.StyleDefault.Dim(true)
			},
		},
		{
			name:  "RGB",
			input: []int{38, 2, 1, 2, 3},
			expected: func() tcell.Style {
				color := tcell.NewRGBColor(1, 2, 3)
				return tcell.StyleDefault.Foreground(color)
			},
		},
		{
			name:  "RGB fg and bg",
			input: []int{38, 2, 1, 2, 3, 48, 2, 1, 2, 3},
			expected: func() tcell.Style {
				color := tcell.NewRGBColor(1, 2, 3)
				return tcell.StyleDefault.Foreground(color).Background(color)
			},
		},
		{
			name:  "RGB with colorspace",
			input: []int{38, 2, 0, 1, 2, 3},
			expected: func() tcell.Style {
				color := tcell.NewRGBColor(1, 2, 3)
				return tcell.StyleDefault.Foreground(color)
			},
		},
		{
			name:  "RGB fg and bg with colorspace",
			input: []int{38, 2, 0, 1, 2, 3, 48, 2, 0, 1, 2, 3},
			expected: func() tcell.Style {
				color := tcell.NewRGBColor(1, 2, 3)
				return tcell.StyleDefault.Foreground(color).Background(color)
			},
		},
		{
			name:  "256 Color",
			input: []int{38, 5, 0},
			expected: func() tcell.Style {
				color := tcell.PaletteColor(0)
				return tcell.StyleDefault.Foreground(color)
			},
		},
		{
			name:  "256 with extra params",
			input: []int{38, 5, 0, 0, 0, 0, 0},
			expected: func() tcell.Style {
				return tcell.StyleDefault
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vt := New()

			vt.sgr(test.input)
			assert.Equal(t, test.expected(), vt.cursor.attrs)
		})
	}
}
