package termutil

type Sixel struct {
	X    uint16
	Y    uint64 // raw line
	Data []byte
}

type VisibleSixel struct {
	ViewLineOffset int
	Sixel          Sixel
}

func (b *Buffer) addSixel(data []byte) {
	b.sixels = append(b.sixels, Sixel{
		X:    b.CursorColumn(),
		Y:    b.cursorPosition.Line,
		Data: data,
	})
}

func (b *Buffer) GetVisibleSixels() []VisibleSixel {

	firstLine := b.convertViewLineToRawLine(0)
	lastLine := b.convertViewLineToRawLine(b.viewHeight - 1)

	var visible []VisibleSixel

	for _, sixelImage := range b.sixels {
		if sixelImage.Y < firstLine {
			continue
		}
		if sixelImage.Y > lastLine {
			continue
		}

		visible = append(visible, VisibleSixel{
			ViewLineOffset: int(sixelImage.Y) - int(firstLine),
			Sixel:          sixelImage,
		})
	}

	return visible
}

func (t *Terminal) handleSixel(readChan chan MeasuredRune) (renderRequired bool) {

	var data []rune

	var inEscape bool

	for {
		r := <-readChan

		switch r.Rune {
		case 0x1b:
			inEscape = true
			continue
		case 0x5c:
			if inEscape {
				t.activeBuffer.addSixel([]byte(string(data)))
				return true
			}
		}

		inEscape = false

		data = append(data, r.Rune)
	}
}
