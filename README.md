# tcell-term

A virtual terminal widget for [tcell](https://github.com/gdamore/tcell/)

```go
s, err := tcell.NewScreen()
if err != nil {
	panic(err)
}
quit := make(chan struct{})
termRedraw := make(chan struct{})

w, h := s.Size()
cmd := exec.Cmd("less", "/etc/hosts")
termWidth, termHeight := w / 2, h / 2
termX, termY := w / 4, h / 4
term := tcellterm.New()

//run command in term
go func() {
	term.Run(cmd, termRedraw, termWidth, termHeight)
	cmd.Wait()
	quit <- struct{}{}
}()

//event loop
go func() {
	for {
		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			//send key events to the terminal
			term.Event(ev)
		case *tcell.EventResize:
			w, h := s.Size()
			lh := h / 2
			lw := w / 2
			//resize event for the terminal
			term.Resize(lw, lh)
			s.Sync()
		}
	}
}()

//draw loop
loop:
for {
	select {
	case <-quit:
		break loop
	//terminal wants to be redrawn
	case <-termRedraw:
	}
	term.Draw(s, termX, termY)
}

s.Fini()
```
