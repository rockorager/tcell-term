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
	topMargin             int // see DECSTBM docs - this is for scrollable regions
	bottomMargin          int // see DECSTBM docs - this is for scrollable regions
	viewWidth             int
	viewHeight            int
	cursorPosition        *position // raw
	cursorAttr            tcell.Style
	scrollLinesFromBottom int
	maxLines              int
	tabStops              []int
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
	Line int
	Col  int
}

// newBuffer creates a new terminal buffer
func newBuffer(width, height int, maxLines int, fg tcell.Color, bg tcell.Color) *buffer {
	b := &buffer{
		lines:        []line{},
		viewHeight:   height,
		viewWidth:    width,
		maxLines:     maxLines,
		topMargin:    0,
		bottomMargin: height - 1,
		cursorAttr:   tcell.StyleDefault,
		charsets:     []*map[rune]rune{nil, nil},
		modes: modes{
			LineFeedMode: true,
			AutoWrap:     true,
			ShowCursor:   true,
		},
		cursorShape:    tcell.CursorStyleDefault,
		cursorPosition: &position{},
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
	return (b.topMargin > 0) || (b.bottomMargin < (b.ViewHeight() - 1))
}

func (b *buffer) inScrollableRegion() bool {
	cursorVY := b.convertRawLineToViewLine(b.cursorPosition.Line)
	return b.hasScrollableRegion() && cursorVY >= b.topMargin && cursorVY <= b.bottomMargin
}

// NOTE: bottom is exclusive
func (b *buffer) getAreaScrollRange() (top int, bottom int) {
	top = b.convertViewLineToRawLine(b.topMargin)
	bottom = b.convertViewLineToRawLine(b.bottomMargin) + 1
	if bottom > len(b.lines) {
		bottom = len(b.lines)
	}
	return top, bottom
}

func (b *buffer) areaScrollDown(lines int) {
	// NOTE: bottom is exclusive
	top, bottom := b.getAreaScrollRange()

	for i := bottom; i > top; {
		i--
		if i >= top+lines {
			b.lines[i] = b.lines[i-lines]
		} else {
			b.lines[i] = b.defaultLine()
		}
	}
}

func (b *buffer) areaScrollUp(lines int) {
	// NOTE: bottom is exclusive
	top, bottom := b.getAreaScrollRange()

	for i := top; i < bottom; i++ {
		from := i + lines
		if from < bottom {
			b.lines[i] = b.lines[from]
		} else {
			b.lines[i] = b.defaultLine()
		}
	}
}

func (b *buffer) saveCursor() {
	copiedAttr := b.cursorAttr
	b.savedCursorAttr = &copiedAttr
	b.savedCursorPos = *b.cursorPosition
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
	*b.cursorPosition = b.savedCursorPos
	if b.savedCharsets != nil {
		b.charsets = make([]*map[rune]rune, len(b.savedCharsets))
		copy(b.charsets, b.savedCharsets)
		b.currentCharset = b.savedCurrentCharset
	}
}

func (b *buffer) getCursorAttr() *tcell.Style {
	return &b.cursorAttr
}

func (b *buffer) getCell(viewCol int, viewRow int) *cell {
	rawLine := b.convertViewLineToRawLine(viewRow)
	return b.getRawCell(viewCol, rawLine)
}

func (b *buffer) getRawCell(viewCol int, rawLine int) *cell {
	if rawLine >= len(b.lines) {
		return nil
	}
	line := &b.lines[rawLine]
	if viewCol >= len(line.cells) {
		return nil
	}
	return &line.cells[viewCol]
}

// Column returns cursor column
func (b *buffer) cursorColumn() int {
	// @todo originMode and left margin
	return b.cursorPosition.Col
}

// cursorLineAbsolute returns absolute cursor line coordinate (ignoring Origin Mode) - view format
func (b *buffer) cursorLineAbsolute() int {
	cursorVY := b.convertRawLineToViewLine(b.cursorPosition.Line)
	return cursorVY
}

// cursorLine returns cursor line (in Origin Mode it is relative to the top margin)
func (b *buffer) cursorLine() int {
	if b.modes.OriginMode {
		return b.cursorLineAbsolute() - b.topMargin
	}
	return b.cursorLineAbsolute()
}

// cursor Y (raw)
func (b *buffer) RawLine() int {
	return b.cursorPosition.Line
}

func (b *buffer) convertViewLineToRawLine(viewLine int) int {
	rawHeight := b.Height()
	if b.viewHeight > rawHeight {
		return viewLine
	}
	return viewLine + rawHeight - (b.viewHeight + b.scrollLinesFromBottom)
}

func (b *buffer) convertRawLineToViewLine(rawLine int) int {
	rawHeight := b.Height()
	if b.viewHeight > rawHeight {
		return rawLine
	}
	return rawLine - (rawHeight - (b.viewHeight + b.scrollLinesFromBottom))
}

func (b *buffer) GetVPosition() int {
	result := b.Height() - b.ViewHeight() - b.scrollLinesFromBottom
	if result < 0 {
		result = 0
	}

	return result
}

// Width returns the width of the buffer in columns
func (b *buffer) Width() int {
	return b.viewWidth
}

func (b *buffer) ViewWidth() int {
	return b.viewWidth
}

func (b *buffer) Height() int {
	return len(b.lines)
}

func (b *buffer) ViewHeight() int {
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
	index := b.RawLine()
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

		if cursorVY < b.bottomMargin {
			b.cursorPosition.Line++
		} else {
			b.areaScrollUp(1)
		}

		return
	}

	if cursorVY >= b.ViewHeight()-1 {
		b.lines = append(b.lines, b.defaultLine())
		maxLines := b.GetMaxLines()
		if len(b.lines) > maxLines {
			copy(b.lines, b.lines[len(b.lines)-maxLines:])
			b.lines = b.lines[:maxLines]
		}
	}
	b.cursorPosition.Line++
}

func (b *buffer) reverseIndex() {
	cursorVY := b.convertRawLineToViewLine(b.cursorPosition.Line)

	if cursorVY == b.topMargin {
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

			for b.cursorColumn() >= len(line.cells) {
				line.append(b.defaultCell(b.cursorColumn() == len(line.cells)))
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

			for b.cursorColumn() >= len(line.cells) {
				line.append(b.defaultCell(b.cursorColumn() == len(line.cells)))
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
			b.movePosition(b.Width()-1, -1)
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
	b.setPosition(tabStop, b.cursorLine())
}

// return next tab stop x pos
func (b *buffer) getNextTabStopAfter(col int) int {
	defaultStop := col + (tabSize - (col % tabSize))
	if defaultStop == col {
		defaultStop += tabSize
	}

	var low int
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

func (b *buffer) movePosition(x int, y int) {
	var toX int
	var toY int

	if b.cursorColumn()+x < 0 {
		toX = 0
	} else {
		toX = b.cursorColumn() + x
	}

	// should either use CursorLine() and setPosition() or use absolutes, mind Origin Mode (DECOM)
	if b.cursorLine()+y < 0 {
		toY = 0
	} else {
		toY = b.cursorLine() + y
	}

	b.setPosition(toX, toY)
}

func (b *buffer) setPosition(col int, line int) {
	useCol := col
	useLine := line
	maxLine := b.ViewHeight() - 1

	if b.modes.OriginMode {
		useLine += b.topMargin
		maxLine = b.bottomMargin
		// @todo left and right margins
	}
	if useLine > maxLine {
		useLine = maxLine
	}

	if useCol >= b.ViewWidth() {
		useCol = b.ViewWidth() - 1
	}

	l := b.convertViewLineToRawLine(useLine)
	b.cursorPosition.Col = useCol
	b.cursorPosition.Line = l
}

func (b *buffer) GetVisibleLines() []line {
	lines := []line{}

	for i := b.Height() - b.ViewHeight(); i < b.Height(); i++ {
		y := i - b.scrollLinesFromBottom
		if y >= 0 && y < len(b.lines) {
			lines = append(lines, b.lines[y])
		}
	}
	return lines
}

// tested to here

func (b *buffer) clear() {
	for i := 0; i < int(b.ViewHeight()); i++ {
		b.lines = append(b.lines, b.defaultLine())
	}
	b.setPosition(0, 0)
}

// creates if necessary
func (b *buffer) getCurrentLine() *line {
	cursorVY := b.convertRawLineToViewLine(b.cursorPosition.Line)
	return b.getViewLine(cursorVY)
}

func (b *buffer) getViewLine(index int) *line {
	if index >= b.ViewHeight() {
		return &b.lines[len(b.lines)-1]
	}

	if len(b.lines) < b.ViewHeight() {
		for index >= len(b.lines) {
			b.lines = append(b.lines, b.defaultLine())
		}
		return &b.lines[index]
	}

	if raw := b.convertViewLineToRawLine(index); raw < len(b.lines) {
		return &b.lines[raw]
	}

	return nil
}

func (b *buffer) eraseLine() {
	line := b.getCurrentLine()

	for i := 0; i < b.viewWidth; i++ {
		if i >= len(line.cells) {
			line.cells = append(line.cells, b.defaultCell(false))
		} else {
			line.cells[i] = b.defaultCell(false)
		}
	}
}

func (b *buffer) eraseLineToCursor() {
	line := b.getCurrentLine()
	for i := 0; i <= b.cursorPosition.Col; i++ {
		if i < len(line.cells) {
			line.cells[i] = b.defaultCell(false)
		}
	}
}

func (b *buffer) eraseLineFromCursor() {
	line := b.getCurrentLine()

	for i := b.cursorPosition.Col; i < b.viewWidth; i++ {
		if i >= len(line.cells) {
			line.cells = append(line.cells, b.defaultCell(false))
		} else {
			line.cells[i] = b.defaultCell(false)
		}
	}
}

func (b *buffer) eraseDisplay() {
	for y := 0; y < b.ViewHeight(); y++ {
		rawLine := b.convertViewLineToRawLine(y)
		if rawLine < len(b.lines) {
			b.lines[rawLine] = b.defaultLine()
		}
	}
}

func (b *buffer) deleteChars(n int) {
	line := b.getCurrentLine()
	if b.cursorPosition.Col >= len(line.cells) {
		return
	}
	before := line.cells[:b.cursorPosition.Col]
	if b.cursorPosition.Col+n >= len(line.cells) {
		n = len(line.cells) - b.cursorPosition.Col
	}
	after := line.cells[b.cursorPosition.Col+n:]
	line.cells = append(before, after...)
}

func (b *buffer) eraseCharacters(n int) {
	line := b.getCurrentLine()

	max := b.cursorPosition.Col + n
	if max > len(line.cells) {
		max = len(line.cells)
	}

	for i := b.cursorPosition.Col; i < max; i++ {
		// TODO should this be default or blank?
		line.cells[i] = b.blankCell()
	}
}

func (b *buffer) eraseDisplayFromCursor() {
	line := b.getCurrentLine()

	pos := b.cursorPosition.Col
	if pos > len(line.cells) {
		pos = len(line.cells)
	}
	for i := pos; i < len(line.cells); i++ {
		line.cells[i] = b.defaultCell(false)
	}

	for rawLine := b.cursorPosition.Line + 1; rawLine < len(b.lines); rawLine++ {
		b.lines[rawLine] = b.defaultLine()
	}
}

func (b *buffer) eraseDisplayToCursor() {
	line := b.getCurrentLine()

	for i := 0; i <= b.cursorPosition.Col; i++ {
		if i >= len(line.cells) {
			break
		}
		line.cells[i] = b.defaultCell(false)
	}

	cursorVY := b.convertRawLineToViewLine(b.cursorPosition.Line)

	for i := 0; i < cursorVY; i++ {
		rawLine := b.convertViewLineToRawLine(i)
		if rawLine < len(b.lines) {
			b.lines[rawLine] = b.defaultLine()
		}
	}
}

func (b *buffer) GetMaxLines() int {
	result := b.maxLines
	if result < b.viewHeight {
		result = b.viewHeight
	}

	return result
}

func (b *buffer) setVerticalMargins(top int, bottom int) {
	b.topMargin = top
	b.bottomMargin = bottom
}

// resetVerticalMargins resets margins to extreme positions
func (b *buffer) resetVerticalMargins(height int) {
	b.setVerticalMargins(0, height-1)
}

// defaultCell returns a cell with default styling
func (b *buffer) defaultCell(applyEffects bool) cell {
	attr := tcell.StyleDefault
	if !applyEffects {
		attr = attr.Blink(false)
		attr = attr.Bold(false)
		attr = attr.Dim(false)
		attr = attr.Reverse(false)
		attr = attr.Underline(false)
		attr = attr.Dim(false)
	}
	return cell{attr: attr}
}

// defaultLine returns a line of empty cells with default styling
func (b *buffer) defaultLine() line {
	cells := []cell{}
	for x := 0; x < b.ViewWidth(); x++ {
		cell := b.defaultCell(false)
		cell.setRune(measuredRune{
			rune:  ' ',
			width: 1,
		})
		cells = append(cells, cell)
	}
	return line{
		wrapped: false,
		cells:   cells,
	}
}

func (b *buffer) IsNewLineMode() bool {
	return !b.modes.LineFeedMode
}

func (b *buffer) tabReset() {
	b.tabStops = nil
}

func (b *buffer) tabSet(index int) {
	b.tabStops = append(b.tabStops, index)
}

func (b *buffer) tabClear(index int) {
	var filtered []int
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

func (b *buffer) resizeView(width int, height int) {
	b.viewWidth = width
	b.viewHeight = height
	// scroll to bottom
	b.scrollLinesFromBottom = 0
	b.resetVerticalMargins(b.viewHeight)
}

func (b *buffer) GetScrollOffset() int {
	return b.scrollLinesFromBottom
}

func (b *buffer) SetScrollOffset(offset int) {
	b.scrollLinesFromBottom = offset
}

func (b *buffer) ScrollToEnd() {
	b.scrollLinesFromBottom = 0
}

func (b *buffer) ScrollUp(lines int) {
	if b.scrollLinesFromBottom+lines < len(b.lines)-b.viewHeight {
		b.scrollLinesFromBottom += lines
	} else {
		lines := len(b.lines) - b.viewHeight
		if lines < 0 {
			lines = 0
		}
		b.scrollLinesFromBottom = lines
	}
}

func (b *buffer) ScrollDown(lines int) {
	if b.scrollLinesFromBottom-lines >= 0 {
		b.scrollLinesFromBottom -= lines
	} else {
		b.scrollLinesFromBottom = 0
	}
}

// blankCell returns an empty cells with the current cursor style
func (b *buffer) blankCell() cell {
	cell := b.defaultCell(false)
	cell.setStyle(*b.getCursorAttr())
	cell.setRune(measuredRune{
		rune:  ' ',
		width: 1,
	})
	return cell
}

// blankLine returns a line of empty cells with the current cursor style
func (b *buffer) blankLine() line {
	cells := []cell{}
	for x := 0; x < b.ViewWidth(); x++ {
		cell := b.blankCell()
		cells = append(cells, cell)
	}
	return line{
		wrapped: false,
		cells:   cells,
	}
}
