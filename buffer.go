package tcellterm

import (
	"image"
	"sync"

	"github.com/gdamore/tcell/v2"
)

const tabSize = 8

type buffer struct {
	lines                 []line
	savedCursorPos        position
	savedCursorAttr       *tcell.Style
	cursorShape           tcell.CursorStyle
	savedCharsets         []*map[rune]rune
	savedCurrentCharset   int
	topMargin             uint // see DECSTBM docs - this is for scrollable regions
	bottomMargin          uint // see DECSTBM docs - this is for scrollable regions
	viewWidth             uint16
	viewHeight            uint16
	cursorPosition        position // raw
	cursorAttr            tcell.Style
	scrollLinesFromBottom uint
	maxLines              uint64
	tabStops              []uint16
	charsets              []*map[rune]rune // array of 2 charsets, nil means ASCII (no conversion)
	currentCharset        int              // active charset index in charsets array, valid values are 0 or 1
	modes                 modes
	selectionStart        *position
	selectionEnd          *position
	highlightStart        *position
	highlightEnd          *position
	highlightAnnotation   *annotation
	sixels                []sixel
	selectionMu           sync.Mutex
}

type annotation struct {
	Image  image.Image
	Text   string
	Width  float64 // Width in cells
	Height float64 // Height in cells
}

type selection struct {
	Start position
	End   position
}

type position struct {
	Line uint64
	Col  uint16
}

// newBuffer creates a new terminal buffer
func newBuffer(width, height uint16, maxLines uint64, fg tcell.Color, bg tcell.Color) *buffer {
	b := &buffer{
		lines:        []line{},
		viewHeight:   height,
		viewWidth:    width,
		maxLines:     maxLines,
		topMargin:    0,
		bottomMargin: uint(height - 1),
		cursorAttr:   tcell.StyleDefault,
		charsets:     []*map[rune]rune{nil, nil},
		modes: modes{
			LineFeedMode: true,
			AutoWrap:     true,
			ShowCursor:   true,
		},
		cursorShape: tcell.CursorStyleDefault,
	}
	return b
}

func (b *buffer) setCursorShape(shape tcell.CursorStyle) {
	b.cursorShape = shape
}

func (b *buffer) getCursorShape() tcell.CursorStyle {
	return b.cursorShape
}

func (b *buffer) isCursorVisible() bool {
	return b.modes.ShowCursor
}

func (b *buffer) isApplicationCursorKeysModeEnabled() bool {
	return b.modes.ApplicationCursorKeys
}

func (b *buffer) hasScrollableRegion() bool {
	return b.topMargin > 0 || b.bottomMargin < uint(b.ViewHeight())-1
}

func (b *buffer) inScrollableRegion() bool {
	cursorVY := b.convertRawLineToViewLine(b.cursorPosition.Line)
	return b.hasScrollableRegion() && uint(cursorVY) >= b.topMargin && uint(cursorVY) <= b.bottomMargin
}

// NOTE: bottom is exclusive
func (b *buffer) getAreaScrollRange() (top uint64, bottom uint64) {
	top = b.convertViewLineToRawLine(uint16(b.topMargin))
	bottom = b.convertViewLineToRawLine(uint16(b.bottomMargin)) + 1
	if bottom > uint64(len(b.lines)) {
		bottom = uint64(len(b.lines))
	}
	return top, bottom
}

func (b *buffer) areaScrollDown(lines uint16) {
	// NOTE: bottom is exclusive
	top, bottom := b.getAreaScrollRange()

	for i := bottom; i > top; {
		i--
		if i >= top+uint64(lines) {
			b.lines[i] = b.lines[i-uint64(lines)]
			b.lines[i].setDirty(true)
		} else {
			b.lines[i] = newLine()
		}
	}
}

func (b *buffer) areaScrollUp(lines uint16) {
	// NOTE: bottom is exclusive
	top, bottom := b.getAreaScrollRange()

	for i := top; i < bottom; i++ {
		from := i + uint64(lines)
		if from < bottom {
			b.lines[i] = b.lines[from]
			b.lines[i].setDirty(true)
		} else {
			b.lines[i] = newLine()
		}
	}
}

func (b *buffer) saveCursor() {
	copiedAttr := b.cursorAttr
	b.savedCursorAttr = &copiedAttr
	b.savedCursorPos = b.cursorPosition
	b.savedCharsets = make([]*map[rune]rune, len(b.charsets))
	copy(b.savedCharsets, b.charsets)
	b.savedCurrentCharset = b.currentCharset
}

func (b *buffer) restoreCursor() {
	// TODO: Do we need to restore attributes on cursor restore? conflicting sources but vim + htop work better without doing so
	//if buffer.savedCursorAttr != nil {
	// copiedAttr := *buffer.savedCursorAttr
	// copiedAttr.bgColour = buffer.defaultCell(false).attr.bgColour
	// copiedAttr.fgColour = buffer.defaultCell(false).attr.fgColour
	// buffer.cursorAttr = copiedAttr
	//}
	b.cursorPosition = b.savedCursorPos
	if b.savedCharsets != nil {
		b.charsets = make([]*map[rune]rune, len(b.savedCharsets))
		copy(b.charsets, b.savedCharsets)
		b.currentCharset = b.savedCurrentCharset
	}
}

func (b *buffer) getCursorAttr() *tcell.Style {
	return &b.cursorAttr
}

func (b *buffer) getCell(viewCol uint16, viewRow uint16) *cell {
	rawLine := b.convertViewLineToRawLine(viewRow)
	return b.getRawCell(viewCol, rawLine)
}

func (b *buffer) getRawCell(viewCol uint16, rawLine uint64) *cell {
	if rawLine >= uint64(len(b.lines)) {
		return nil
	}
	line := &b.lines[rawLine]
	if int(viewCol) >= len(line.cells) {
		return nil
	}
	return &line.cells[viewCol]
}

// Column returns cursor column
func (b *buffer) cursorColumn() uint16 {
	// @todo originMode and left margin
	return b.cursorPosition.Col
}

// cursorLineAbsolute returns absolute cursor line coordinate (ignoring Origin Mode) - view format
func (b *buffer) cursorLineAbsolute() uint16 {
	cursorVY := b.convertRawLineToViewLine(b.cursorPosition.Line)
	return cursorVY
}

// cursorLine returns cursor line (in Origin Mode it is relative to the top margin)
func (b *buffer) cursorLine() uint16 {
	if b.modes.OriginMode {
		return b.cursorLineAbsolute() - uint16(b.topMargin)
	}
	return b.cursorLineAbsolute()
}

// cursor Y (raw)
func (b *buffer) RawLine() uint64 {
	return b.cursorPosition.Line
}

func (b *buffer) convertViewLineToRawLine(viewLine uint16) uint64 {
	rawHeight := b.Height()
	if int(b.viewHeight) > rawHeight {
		return uint64(viewLine)
	}
	return uint64(int(viewLine) + (rawHeight - int(b.viewHeight+uint16(b.scrollLinesFromBottom))))
}

func (b *buffer) convertRawLineToViewLine(rawLine uint64) uint16 {
	rawHeight := b.Height()
	if int(b.viewHeight) > rawHeight {
		return uint16(rawLine)
	}
	return uint16(int(rawLine) - (rawHeight - int(b.viewHeight+uint16(b.scrollLinesFromBottom))))
}

func (b *buffer) GetVPosition() int {
	result := int(uint(b.Height()) - uint(b.ViewHeight()) - b.scrollLinesFromBottom)
	if result < 0 {
		result = 0
	}

	return result
}

// Width returns the width of the buffer in columns
func (b *buffer) Width() uint16 {
	return b.viewWidth
}

func (b *buffer) ViewWidth() uint16 {
	return b.viewWidth
}

func (b *buffer) Height() int {
	return len(b.lines)
}

func (b *buffer) ViewHeight() uint16 {
	return b.viewHeight
}

func (b *buffer) deleteLine() {
	// see
	// https://github.com/james4k/terminal/blob/b4bcb6ee7c08ae4930eecdeb1ba90073c5f40d71/state.go#L682
	b.areaScrollUp(1)
}

func (b *buffer) insertLine() {
	// see
	// https://github.com/james4k/terminal/blob/b4bcb6ee7c08ae4930eecdeb1ba90073c5f40d71/state.go#L682
	b.areaScrollDown(1)
}

func (b *buffer) insertBlankCharacters(count int) {
	index := int(b.RawLine())
	for i := 0; i < count; i++ {
		cells := b.lines[index].cells
		b.lines[index].cells = append(cells[:b.cursorPosition.Col], append([]cell{b.defaultCell(true)}, cells[b.cursorPosition.Col:]...)...)
	}
}

func (b *buffer) insertLines(count int) {
	if b.hasScrollableRegion() && !b.inScrollableRegion() {
		// should have no effect outside of scrollable region
		return
	}

	b.cursorPosition.Col = 0

	for i := 0; i < count; i++ {
		b.insertLine()
	}
}

func (b *buffer) deleteLines(count int) {
	if b.hasScrollableRegion() && !b.inScrollableRegion() {
		// should have no effect outside of scrollable region
		return
	}

	b.cursorPosition.Col = 0

	for i := 0; i < count; i++ {
		b.deleteLine()
	}
}

func (b *buffer) index() {
	// This sequence causes the active position to move downward one line without changing the column position.
	// If the active position is at the bottom margin, a scroll up is performed."

	cursorVY := b.convertRawLineToViewLine(b.cursorPosition.Line)

	if b.inScrollableRegion() {

		if uint(cursorVY) < b.bottomMargin {
			b.cursorPosition.Line++
		} else {
			b.areaScrollUp(1)
		}

		return
	}

	if cursorVY >= b.ViewHeight()-1 {
		b.lines = append(b.lines, newLine())
		maxLines := b.GetMaxLines()
		if uint64(len(b.lines)) > maxLines {
			copy(b.lines, b.lines[uint64(len(b.lines))-maxLines:])
			b.lines = b.lines[:maxLines]
			for _, line := range b.lines {
				line.setDirty(true)
			}
		}
	}
	b.cursorPosition.Line++
}

func (b *buffer) reverseIndex() {
	cursorVY := b.convertRawLineToViewLine(b.cursorPosition.Line)

	if uint(cursorVY) == b.topMargin {
		b.areaScrollDown(1)
	} else if cursorVY > 0 {
		b.cursorPosition.Line--
	}
}

// write will write a rune to the terminal at the position of the cursor, and increment the cursor position
func (b *buffer) write(runes ...measuredRune) {
	// scroll to bottom on input
	b.scrollLinesFromBottom = 0

	for _, r := range runes {

		line := b.getCurrentLine()

		if b.modes.ReplaceMode {

			if b.cursorColumn() >= b.Width() {
				if b.modes.AutoWrap {
					b.cursorPosition.Line++
					b.cursorPosition.Col = 0
					line = b.getCurrentLine()

				} else {
					// no more room on line and wrapping is disabled
					return
				}
			}

			for int(b.cursorColumn()) >= len(line.cells) {
				line.append(b.defaultCell(int(b.cursorColumn()) == len(line.cells)))
			}
			line.cells[b.cursorPosition.Col].attr = b.cursorAttr
			line.cells[b.cursorPosition.Col].setRune(r)
			b.incrementCursorPosition()
			continue
		}

		if b.cursorColumn() >= b.Width() { // if we're after the line, move to next

			if b.modes.AutoWrap {

				b.newLineEx(true)

				newLine := b.getCurrentLine()
				if len(newLine.cells) == 0 {
					newLine.append(b.defaultCell(true))
				}
				cell := &newLine.cells[0]
				cell.setRune(r)
				cell.attr = b.cursorAttr

			} else {
				// no more room on line and wrapping is disabled
				return
			}
		} else {

			for int(b.cursorColumn()) >= len(line.cells) {
				line.append(b.defaultCell(int(b.cursorColumn()) == len(line.cells)))
			}

			cell := &line.cells[b.cursorColumn()]
			cell.setRune(r)
			cell.attr = b.cursorAttr
		}

		b.incrementCursorPosition()
	}
}

func (b *buffer) incrementCursorPosition() {
	// we can increment one column past the end of the line.
	// this is effectively the beginning of the next line, except when we \r etc.
	if b.cursorColumn() < b.Width() {
		b.cursorPosition.Col++
	}
}

func (b *buffer) inDoWrap() bool {
	// xterm uses 'do_wrap' flag for this special terminal state
	// we use the cursor position right after the boundary
	// let's see how it works out
	return b.cursorPosition.Col == b.viewWidth // @todo rightMargin
}

func (b *buffer) backspace() {
	if b.cursorPosition.Col == 0 {
		line := b.getCurrentLine()
		if line.wrapped {
			b.movePosition(int16(b.Width()-1), -1)
		}
	} else if b.inDoWrap() {
		// the "do_wrap" implementation
		b.movePosition(-2, 0)
	} else {
		b.movePosition(-1, 0)
	}
}

func (b *buffer) carriageReturn() {
	cursorVY := b.convertRawLineToViewLine(b.cursorPosition.Line)

	for {
		line := b.getCurrentLine()
		if line == nil {
			break
		}
		if line.wrapped && cursorVY > 0 {
			b.cursorPosition.Line--
		} else {
			break
		}
	}

	b.cursorPosition.Col = 0
}

func (b *buffer) tab() {
	tabStop := b.getNextTabStopAfter(b.cursorPosition.Col)
	for b.cursorPosition.Col < tabStop && b.cursorPosition.Col < b.viewWidth-1 { // @todo rightMargin
		b.write(measuredRune{rune: ' ', width: 1})
	}
}

// return next tab stop x pos
func (b *buffer) getNextTabStopAfter(col uint16) uint16 {
	defaultStop := col + (tabSize - (col % tabSize))
	if defaultStop == col {
		defaultStop += tabSize
	}

	var low uint16
	for _, stop := range b.tabStops {
		if stop > col {
			if stop < low || low == 0 {
				low = stop
			}
		}
	}

	if low == 0 {
		return defaultStop
	}

	return low
}

func (b *buffer) newLine() {
	b.newLineEx(false)
}

func (b *buffer) verticalTab() {
	b.index()

	for {
		line := b.getCurrentLine()
		if !line.wrapped {
			break
		}
		b.index()
	}
}

func (b *buffer) newLineEx(forceCursorToMargin bool) {
	if b.IsNewLineMode() || forceCursorToMargin {
		b.cursorPosition.Col = 0
	}
	b.index()

	for {
		line := b.getCurrentLine()
		if !line.wrapped {
			break
		}
		b.index()
	}
}

func (b *buffer) movePosition(x int16, y int16) {
	var toX uint16
	var toY uint16

	if int16(b.cursorColumn())+x < 0 {
		toX = 0
	} else {
		toX = uint16(int16(b.cursorColumn()) + x)
	}

	// should either use CursorLine() and setPosition() or use absolutes, mind Origin Mode (DECOM)
	if int16(b.cursorLine())+y < 0 {
		toY = 0
	} else {
		toY = uint16(int16(b.cursorLine()) + y)
	}

	b.setPosition(toX, toY)
}

func (b *buffer) setPosition(col uint16, line uint16) {
	useCol := col
	useLine := line
	maxLine := b.ViewHeight() - 1

	if b.modes.OriginMode {
		useLine += uint16(b.topMargin)
		maxLine = uint16(b.bottomMargin)
		// @todo left and right margins
	}
	if useLine > maxLine {
		useLine = maxLine
	}

	if useCol >= b.ViewWidth() {
		useCol = b.ViewWidth() - 1
	}

	b.cursorPosition.Col = useCol
	b.cursorPosition.Line = b.convertViewLineToRawLine(useLine)
}

func (b *buffer) GetVisibleLines() []line {
	lines := []line{}

	for i := b.Height() - int(b.ViewHeight()); i < b.Height(); i++ {
		y := i - int(b.scrollLinesFromBottom)
		if y >= 0 && y < len(b.lines) {
			lines = append(lines, b.lines[y])
		}
	}
	return lines
}

// tested to here

func (b *buffer) clear() {
	for i := 0; i < int(b.ViewHeight()); i++ {
		b.lines = append(b.lines, newLine())
	}
	b.setPosition(0, 0)
}

// creates if necessary
func (b *buffer) getCurrentLine() *line {
	cursorVY := b.convertRawLineToViewLine(b.cursorPosition.Line)
	return b.getViewLine(cursorVY)
}

func (b *buffer) getViewLine(index uint16) *line {
	if index >= b.ViewHeight() {
		return &b.lines[len(b.lines)-1]
	}

	if len(b.lines) < int(b.ViewHeight()) {
		for int(index) >= len(b.lines) {
			b.lines = append(b.lines, newLine())
		}
		return &b.lines[int(index)]
	}

	if raw := int(b.convertViewLineToRawLine(index)); raw < len(b.lines) {
		return &b.lines[raw]
	}

	return nil
}

func (b *buffer) eraseLine() {
	line := b.getCurrentLine()

	for i := 0; i < int(b.viewWidth); i++ {
		if i >= len(line.cells) {
			line.cells = append(line.cells, b.defaultCell(false))
		} else {
			line.cells[i] = b.defaultCell(false)
		}
	}
}

func (b *buffer) eraseLineToCursor() {
	line := b.getCurrentLine()
	_, bg, _ := b.cursorAttr.Decompose()
	for i := 0; i <= int(b.cursorPosition.Col); i++ {
		if i < len(line.cells) {
			line.cells[i].erase(bg)
		}
	}
}

func (b *buffer) eraseLineFromCursor() {
	line := b.getCurrentLine()

	for i := b.cursorPosition.Col; i < b.viewWidth; i++ {
		if int(i) >= len(line.cells) {
			line.cells = append(line.cells, b.defaultCell(false))
		} else {
			line.cells[i] = b.defaultCell(false)
		}
	}
}

func (b *buffer) eraseDisplay() {
	for i := uint16(0); i < (b.ViewHeight()); i++ {
		rawLine := b.convertViewLineToRawLine(i)
		if int(rawLine) < len(b.lines) {
			b.lines[int(rawLine)].cells = []cell{}
		}
	}
}

func (b *buffer) deleteChars(n int) {
	line := b.getCurrentLine()
	if int(b.cursorPosition.Col) >= len(line.cells) {
		return
	}
	before := line.cells[:b.cursorPosition.Col]
	if int(b.cursorPosition.Col)+n >= len(line.cells) {
		n = len(line.cells) - int(b.cursorPosition.Col)
	}
	after := line.cells[int(b.cursorPosition.Col)+n:]
	line.cells = append(before, after...)
}

func (b *buffer) eraseCharacters(n int) {
	line := b.getCurrentLine()

	max := int(b.cursorPosition.Col) + n
	if max > len(line.cells) {
		max = len(line.cells)
	}

	_, bg, _ := b.cursorAttr.Decompose()
	for i := int(b.cursorPosition.Col); i < max; i++ {
		line.cells[i].erase(bg)
	}
}

func (b *buffer) eraseDisplayFromCursor() {
	line := b.getCurrentLine()

	max := int(b.cursorPosition.Col)
	if max > len(line.cells) {
		max = len(line.cells)
	}

	line.cells = line.cells[:max]

	for rawLine := b.cursorPosition.Line + 1; int(rawLine) < len(b.lines); rawLine++ {
		b.lines[int(rawLine)].cells = []cell{}
	}
}

func (b *buffer) eraseDisplayToCursor() {
	line := b.getCurrentLine()

	_, bg, _ := b.cursorAttr.Decompose()
	for i := 0; i <= int(b.cursorPosition.Col); i++ {
		if i >= len(line.cells) {
			break
		}
		line.cells[i].erase(bg)
	}

	cursorVY := b.convertRawLineToViewLine(b.cursorPosition.Line)

	for i := uint16(0); i < cursorVY; i++ {
		rawLine := b.convertViewLineToRawLine(i)
		if int(rawLine) < len(b.lines) {
			b.lines[int(rawLine)].cells = []cell{}
		}
	}
}

func (b *buffer) GetMaxLines() uint64 {
	result := b.maxLines
	if result < uint64(b.viewHeight) {
		result = uint64(b.viewHeight)
	}

	return result
}

func (b *buffer) setVerticalMargins(top uint, bottom uint) {
	b.topMargin = top
	b.bottomMargin = bottom
}

// resetVerticalMargins resets margins to extreme positions
func (b *buffer) resetVerticalMargins(height uint) {
	b.setVerticalMargins(0, height-1)
}

func (b *buffer) defaultCell(applyEffects bool) cell {
	attr := b.cursorAttr
	if !applyEffects {
		attr = attr.Blink(false)
		attr = attr.Bold(false)
		attr = attr.Dim(false)
		attr = attr.Reverse(false)
		attr = attr.Underline(false)
		attr = attr.Dim(false)
	}
	return cell{attr: attr, dirty: true}
}

func (b *buffer) IsNewLineMode() bool {
	return !b.modes.LineFeedMode
}

func (b *buffer) tabReset() {
	b.tabStops = nil
}

func (b *buffer) tabSet(index uint16) {
	b.tabStops = append(b.tabStops, index)
}

func (b *buffer) tabClear(index uint16) {
	var filtered []uint16
	for _, stop := range b.tabStops {
		if stop != b.cursorPosition.Col {
			filtered = append(filtered, stop)
		}
	}
	b.tabStops = filtered
}

func (b *buffer) IsTabSetAtCursor() bool {
	if b.cursorPosition.Col%tabSize > 0 {
		return false
	}
	for _, stop := range b.tabStops {
		if stop == b.cursorPosition.Col {
			return true
		}
	}
	return false
}

func (b *buffer) tabClearAtCursor() {
	b.tabClear(b.cursorPosition.Col)
}

func (b *buffer) tabSetAtCursor() {
	b.tabSet(b.cursorPosition.Col)
}

func (b *buffer) GetScrollOffset() uint {
	return b.scrollLinesFromBottom
}

func (b *buffer) SetScrollOffset(offset uint) {
	b.scrollLinesFromBottom = offset
}

func (b *buffer) ScrollToEnd() {
	b.scrollLinesFromBottom = 0
}

func (b *buffer) ScrollUp(lines uint) {
	if int(b.scrollLinesFromBottom)+int(lines) < len(b.lines)-int(b.viewHeight) {
		b.scrollLinesFromBottom += lines
	} else {
		lines := len(b.lines) - int(b.viewHeight)
		if lines < 0 {
			lines = 0
		}
		b.scrollLinesFromBottom = uint(lines)
	}
}

func (b *buffer) ScrollDown(lines uint) {
	if int(b.scrollLinesFromBottom)-int(lines) >= 0 {
		b.scrollLinesFromBottom -= lines
	} else {
		b.scrollLinesFromBottom = 0
	}
}
