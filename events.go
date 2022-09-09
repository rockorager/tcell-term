package tcellterm

import (
	"time"

	"github.com/gdamore/tcell/v2/views"
)

// EventTitle is emitted when the terminal's title changes
type EventTitle struct {
	when   time.Time
	title  string
	widget *Terminal
}

func (ev *EventTitle) When() time.Time {
	return ev.when
}

func (ev *EventTitle) Widget() views.Widget {
	return ev.widget
}

func (ev *EventTitle) Title() string {
	return ev.title
}
