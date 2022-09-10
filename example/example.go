//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	tcellterm "git.sr.ht/~rockorager/tcell-term"
	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"
)

type model struct {
	term      *tcellterm.Terminal
	s         tcell.Screen
	termView  views.View
	title     *views.TextBar
	titleView views.View
}

func (m *model) HandleEvent(ev tcell.Event) bool {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyCtrlC:
			m.term.Close()
			m.s.Clear()
			m.s.Fini()
			return true
		}
		if m.term != nil {
			return m.term.HandleEvent(ev)
		}
	case *tcell.EventResize:
		if m.term != nil {
			m.termView.Resize(0, 2, -1, -1)
			m.term.Resize()
		}
		m.titleView.Resize(0, 0, -1, 2)
		m.title.Resize()
		m.s.Sync()
		return true
	case *views.EventWidgetContent:
		m.term.Draw()
		m.title.Draw()

		vis, x, y, style := m.term.GetCursor()
		if vis {
			m.s.ShowCursor(x, y+2)
			m.s.SetCursorStyle(style)
		} else {
			m.s.HideCursor()
		}
		m.s.Show()
		return true
	case *tcellterm.EventClosed:
		m.s.Clear()
		m.s.Fini()
		return true
	}
	return false
}

func main() {
	var err error
	f, _ := os.Create("log")
	defer f.Close()
	logbuf := bytes.NewBuffer(nil)
	log.SetOutput(io.MultiWriter(f, logbuf))
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	m := &model{}
	m.s, err = tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if err = m.s.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	m.title = views.NewTextBar()
	m.title.SetCenter(
		"Welcome to tcell-term",
		tcell.StyleDefault.Foreground(tcell.ColorBlue).
			Bold(true).
			Underline(true),
	)

	m.titleView = views.NewViewPort(m.s, 0, 0, -1, 2)
	m.title.Watch(m)
	m.title.SetView(m.titleView)

	m.termView = views.NewViewPort(m.s, 0, 2, -1, -1)
	m.term = tcellterm.New()
	m.term.Watch(m)
	m.term.SetView(m.termView)

	cmd := exec.Command(os.Getenv("SHELL"))
	go func() {
		if err := m.term.Run(cmd); err != nil {
			log.Println(err)
		}
	}()
	for {
		ev := m.s.PollEvent()
		if ev == nil {
			break
		}
		m.HandleEvent(ev)
	}
	os.Stdout.Write(logbuf.Bytes())
}
