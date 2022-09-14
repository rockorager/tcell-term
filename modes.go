package tcellterm

type modes struct {
	ShowCursor            bool
	ApplicationCursorKeys bool
	BlinkingCursor        bool
	ReplaceMode           bool // overwrite character at cursor or insert new
	OriginMode            bool // see DECOM docs - whether cursor is positioned within the margins or not
	LineFeedMode          bool
	ScreenMode            bool // DECSCNM (black on white background)
	AutoWrap              bool
	BracketedPasteMode    bool
}

type (
	mouseMode    uint
	mouseExtMode uint
)

const (
	mouseModeNone mouseMode = iota
	mouseModeX10
	mouseModeVT200
	mouseModeVT200Highlight
	mouseModeButtonEvent
	mouseModeAnyEvent
	mouseExtNone mouseExtMode = iota
	mouseExtUTF
	mouseExtSGR
	mouseExtURXVT
)
