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

	tcellterm "git.sr.ht/~ghost08/tcell-term"
	"github.com/gdamore/tcell/v2"
)

func main() {
	f, _ := os.Create("meh.log")
	defer f.Close()
	logbuf := bytes.NewBuffer(nil)
	log.SetOutput(io.MultiWriter(f, logbuf))
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	s, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if err = s.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	s.Clear()

	quit := make(chan struct{})
	var term *tcellterm.Terminal
	term = tcellterm.New(s, nil)

	cmd := exec.Command("zsh")
	go func() {
		if err := term.Run(cmd); err != nil {
			log.Println(err)
		}
		close(quit)
		s.Fini()
		os.Stdout.Write(logbuf.Bytes())
		os.Exit(0)
	}()
	for {
		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyCtrlC:
				close(quit)
				s.Fini()
				os.Stdout.Write(logbuf.Bytes())
				return
			}
			if term != nil {
				term.HandleEvent(ev)
			}
		case *tcell.EventResize:
			if term != nil {
				term.Resize()
			}
			s.Sync()
		case *tcellterm.RedrawEvent:
			term.Draw()
			vis, x, y, style := term.GetCursor()
			if vis {
				s.ShowCursor(x, y)
				s.SetCursorStyle(style)
			} else {
				s.HideCursor()
			}
			s.Show()
		}

	}
}
