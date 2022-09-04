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
	redraw := make(chan struct{}, 10)
	var term *tcellterm.Terminal
	if term == nil {
		term = tcellterm.New()

		cmd := exec.Command("zsh")
		go func() {
			w, h := s.Size()
			lh := h
			lw := w
			if err := term.Run(cmd, redraw, uint16(lw), uint16(lh)); err != nil {
				log.Println(err)
			}
			s.HideCursor()
			term = nil
			close(quit)
		}()
	}
	go func() {
		for {
			ev := s.PollEvent()
			switch ev := ev.(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyCtrlC:
					close(quit)
					return
				}
				if term != nil {
					term.Event(ev)
				}
			case *tcell.EventResize:
				if term != nil {
					w, h := s.Size()
					lh := h
					lw := w
					term.Resize(lw, lh)
				}
				s.Sync()
			}
		}
	}()

loop:
	for {
		select {
		case <-quit:
			break loop
		case <-redraw:
			term.Draw(s, 0, 0)
			s.Show()
		}
	}

	s.Fini()
	os.Stdout.Write(logbuf.Bytes())
}
