package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"time"

	"github.com/gdamore/tcell/v2"
)

var red = int32(rand.Int() % 256)
var grn = int32(rand.Int() % 256)
var blu = int32(rand.Int() % 256)
var inc = int32(8) // rate of color change
var redi = int32(inc)
var grni = int32(inc)
var blui = int32(inc)
var term *Terminal

func makebox(s tcell.Screen) {
	w, h := s.Size()

	if w == 0 || h == 0 {
		return
	}

	glyphs := []rune{'@', '#', '&', '*', '=', '%', 'Z', 'A'}

	lh := h / 2
	lw := w / 2
	lx := w / 4
	ly := h / 4
	st := tcell.StyleDefault
	gl := ' '

	if s.Colors() == 0 {
		st = st.Reverse(rand.Int()%2 == 0)
		gl = glyphs[rand.Int()%len(glyphs)]
	} else {

		red += redi
		if (red >= 256) || (red < 0) {
			redi = -redi
			red += redi
		}
		grn += grni
		if (grn >= 256) || (grn < 0) {
			grni = -grni
			grn += grni
		}
		blu += blui
		if (blu >= 256) || (blu < 0) {
			blui = -blui
			blu += blui

		}
		st = st.Background(tcell.NewRGBColor(red, grn, blu))
	}
	if term == nil {
		for row := 0; row < lh; row++ {
			for col := 0; col < lw; col++ {
				s.SetCell(lx+col, ly+row, st, gl)
			}
		}
	} else {
		term.Draw(s, 10, 10)
	}
	s.Show()
}

func flipcoin() bool {
	if rand.Int()&1 == 0 {
		return false
	}
	return true
}

func main() {
	f, _ := os.Create("meh.log")
	defer f.Close()
	logbuf := bytes.NewBuffer(nil)
	log.SetOutput(io.MultiWriter(f, logbuf))
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	rand.Seed(time.Now().UnixNano())
	tcell.SetEncodingFallback(tcell.EncodingFallbackASCII)
	s, e := tcell.NewScreen()
	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	if e = s.Init(); e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}

	s.SetStyle(tcell.StyleDefault.
		Foreground(tcell.ColorWhite).
		Background(tcell.ColorBlack))
	s.Clear()

	quit := make(chan struct{})
	redraw := make(chan struct{})
	go func() {
		for {
			ev := s.PollEvent()
			switch ev := ev.(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyEscape:
					close(quit)
					return
				case tcell.KeyEnter:
					if term == nil {
						term = New()
						cmd := exec.Command("less", "terminal.go")
						go func() {
							term.Run(cmd, redraw, 40, 60)
							if err := cmd.Wait(); err != nil {
								log.Println(err)
							}
							term = nil
						}()
						continue
					}
				}
				if term != nil {
					term.Event(ev)
				}
			case *tcell.EventResize:
				s.Sync()
			}
		}
	}()

	cnt := 0
loop:
	for {
		select {
		case <-quit:
			break loop
		case <-time.After(time.Millisecond * 50):
		case <-redraw:
		}
		makebox(s)
		cnt++
		if cnt%(256/int(inc)) == 0 {
			if flipcoin() {
				redi = -redi
			}
			if flipcoin() {
				grni = -grni
			}
			if flipcoin() {
				blui = -blui
			}
		}
	}

	s.Fini()
	os.Stdout.Write(logbuf.Bytes())
}

func colorToString(c tcell.Color) string {
	r, g, b := c.RGB()
	return fmt.Sprintf("%02x%02x%02x", r, g, b)
}
