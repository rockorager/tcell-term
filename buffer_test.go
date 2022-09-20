package tcellterm

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func writeRaw(buf *buffer, runes ...rune) {
	for _, r := range runes {
		buf.write(measuredRune{rune: r, width: 1})
	}
}

func TestBufferCreation(t *testing.T) {
	b := makeBufferForTesting(10, 20)
	assert.Equal(t, 10, b.Width())
	assert.Equal(t, 20, b.ViewHeight())
	assert.Equal(t, 0, b.cursorColumn())
	assert.Equal(t, 0, b.cursorLine())
	assert.NotNil(t, b.lines)
}

func TestOffsets(t *testing.T) {
	b := makeBufferForTesting(10, 3)
	writeRaw(b, []rune("hello")...)
	b.carriageReturn()
	b.newLine()
	writeRaw(b, []rune("hello")...)
	b.carriageReturn()
	b.newLine()
	writeRaw(b, []rune("hello")...)
	b.carriageReturn()
	b.newLine()
	writeRaw(b, []rune("hello")...)
	b.carriageReturn()
	b.newLine()
	writeRaw(b, []rune("hello")...)
	assert.Equal(t, 10, b.ViewWidth())
	assert.Equal(t, 10, b.Width())
	assert.Equal(t, 3, b.ViewHeight())
	assert.Equal(t, 5, b.Height())
}

func TestWritingNewLineAsFirstRuneOnWrappedLine(t *testing.T) {
	b := makeBufferForTesting(3, 20)
	b.modes.LineFeedMode = false

	writeRaw(b, 'a', 'b', 'c')
	assert.Equal(t, 3, b.cursorPosition.Col)
	assert.Equal(t, 0, b.cursorPosition.Line)
	b.newLine()
	assert.Equal(t, 0, b.cursorPosition.Col)
	assert.Equal(t, 1, b.cursorPosition.Line)

	writeRaw(b, 'd', 'e', 'f')
	assert.Equal(t, 3, b.cursorPosition.Col)
	assert.Equal(t, 1, b.cursorPosition.Line)
	b.newLine()

	assert.Equal(t, 0, b.cursorPosition.Col)
	assert.Equal(t, 2, b.cursorPosition.Line)

	require.Equal(t, 3, len(b.lines))
	assert.Equal(t, "abc", b.lines[0].string())
	assert.Equal(t, "def", b.lines[1].string())
}

func TestSetPosition(t *testing.T) {
	b := makeBufferForTesting(120, 80)
	assert.Equal(t, 0, int(b.cursorColumn()))
	assert.Equal(t, 0, int(b.cursorLine()))

	b.setPosition(60, 10)
	assert.Equal(t, 60, int(b.cursorColumn()))
	assert.Equal(t, 10, int(b.cursorLine()))

	b.setPosition(0, 0)
	assert.Equal(t, 0, int(b.cursorColumn()))
	assert.Equal(t, 0, int(b.cursorLine()))

	b.setPosition(120, 90)
	assert.Equal(t, 119, int(b.cursorColumn()))
	assert.Equal(t, 79, int(b.cursorLine()))
}

func TestMovePosition(t *testing.T) {
	b := makeBufferForTesting(120, 80)
	assert.Equal(t, 0, int(b.cursorColumn()))
	assert.Equal(t, 0, int(b.cursorLine()))

	b.movePosition(-1, -1)
	assert.Equal(t, 0, int(b.cursorColumn()))
	assert.Equal(t, 0, int(b.cursorLine()))

	b.movePosition(30, 20)
	assert.Equal(t, 30, int(b.cursorColumn()))
	assert.Equal(t, 20, int(b.cursorLine()))

	b.movePosition(30, 20)
	assert.Equal(t, 60, int(b.cursorColumn()))
	assert.Equal(t, 40, int(b.cursorLine()))

	b.movePosition(-1, -1)
	assert.Equal(t, 59, int(b.cursorColumn()))
	assert.Equal(t, 39, int(b.cursorLine()))

	b.movePosition(100, 100)
	assert.Equal(t, 119, int(b.cursorColumn()))
	assert.Equal(t, 79, int(b.cursorLine()))
}

func TestCarriageReturnOnFullLine(t *testing.T) {
	b := makeBufferForTesting(20, 20)
	writeRaw(b, []rune("abcdeabcdeabcdeabcde")...)
	b.carriageReturn()
	writeRaw(b, []rune("xxxxxxxxxxxxxxxxxxxx")...)
	lines := b.GetVisibleLines()
	assert.Equal(t, "xxxxxxxxxxxxxxxxxxxx", lines[0].string())
}

func TestCarriageReturnOnLineThatDoesntExist(t *testing.T) {
	b := makeBufferForTesting(6, 10)
	b.cursorPosition.Line = 3
	b.carriageReturn()
	assert.Equal(t, 0, b.cursorPosition.Col)
	assert.Equal(t, 3, b.cursorPosition.Line)
}

func TestGetCell(t *testing.T) {
	b := makeBufferForTesting(80, 20)
	writeRaw(b, []rune("Hello")...)
	b.carriageReturn()
	b.newLine()

	writeRaw(b, []rune("there")...)
	b.carriageReturn()
	b.newLine()

	writeRaw(b, []rune("something...")...)
	cell := b.getCell(8, 2)
	require.NotNil(t, cell)
	assert.Equal(t, 'g', cell.rune().rune)
}

func TestGetCellWithHistory(t *testing.T) {
	b := makeBufferForTesting(80, 2)

	writeRaw(b, []rune("Hello")...)
	b.carriageReturn()
	b.newLine()

	writeRaw(b, []rune("there")...)
	b.carriageReturn()
	b.newLine()

	writeRaw(b, []rune("something...")...)

	cell := b.getCell(8, 1)
	require.NotNil(t, cell)
	assert.Equal(t, 'g', cell.rune().rune)
}

func TestGetCellWithBadCursor(t *testing.T) {
	b := makeBufferForTesting(80, 2)
	writeRaw(b, []rune("Hello\r\nthere\r\nsomething...")...)
	require.Nil(t, b.getCell(8, 3))
	require.Nil(t, b.getCell(90, 0))
}

func TestCursorPositionQuerying(t *testing.T) {
	b := makeBufferForTesting(80, 20)
	b.cursorPosition.Col = 17
	b.cursorPosition.Line = 9
	assert.Equal(t, b.cursorPosition.Col, b.cursorColumn())
	assert.Equal(t, b.convertRawLineToViewLine(b.cursorPosition.Line), b.cursorLine())
}

func TestEraseDisplay(t *testing.T) {
	b := makeBufferForTesting(10, 5)
	writeRaw(b, []rune("hello")...)
	b.carriageReturn()
	b.newLine()
	writeRaw(b, []rune("asdasd")...)
	b.carriageReturn()
	b.newLine()
	writeRaw(b, []rune("thing")...)
	b.movePosition(2, 1)
	b.eraseDisplay()
	lines := b.GetVisibleLines()
	for _, line := range lines {
		// Erase display should put in blank characters
		assert.Equal(t, "          ", line.string())
	}
}

func makeBufferForTesting(cols, rows int) *buffer {
	return newBuffer(cols, rows, 100, tcell.ColorWhite, tcell.ColorBlack)
}
