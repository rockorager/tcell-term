package tcellterm

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"
)

// EventTerminal is a generic terminal event which satisfies the
// views.EventWidget interface. It is suitable for embedding in specific
// terminal events.
type EventTerminal struct {
	when   time.Time
	widget *Terminal
}

func newEventTerminal(t *Terminal) *EventTerminal {
	return &EventTerminal{
		when:   time.Now(),
		widget: t,
	}
}

func (ev *EventTerminal) When() time.Time {
	return ev.when
}

func (ev *EventTerminal) Widget() views.Widget {
	return ev.widget
}

// EventClosed is emitted when the terminal exits
type EventClosed struct {
	*EventTerminal
}

// EventTitle is emitted when the terminal's title changes
type EventTitle struct {
	title string

	*EventTerminal
}

func (ev *EventTitle) Title() string {
	return ev.title
}

// EventMouseMode is emitted when the terminal mouse mode changes
type EventMouseMode struct {
	modes []tcell.MouseFlags

	*EventTerminal
}

func (ev *EventMouseMode) Flags() []tcell.MouseFlags {
	return ev.modes
}
