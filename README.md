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

For general discussion or patches, use the [mailing list](https://lists.sr.ht/~ghost08/tcell-term): [~ghost08/tcell-term@lists.sr.ht](mailto:~ghost08/tcell-term@lists.sr.ht).

## Contributing

Anyone can contribute to tcell-term:

-   Clone the repository.
-   Patch the code.
-   Make some tests.
-   Ensure that your code is properly formatted with gofmt.
-   Ensure that everything works as expected.
-   Ensure that you did not break anything.
-   Do not forget to update the docs.

Once you are happy with your work, you can create a commit (or several commits). Follow these general rules:

-   Limit the first line (title) of the commit message to 60 characters.
-   Use a short prefix for the commit title for readability with `git log --oneline`.
-   Use the body of the commit message to actually explain what your patch does and why it is useful.
-   Address only one issue/topic per commit.
-   If you are fixing a ticket, use appropriate [commit trailers](https://man.sr.ht/git.sr.ht/#referencing-tickets-in-git-commit-messages).
-   If you are fixing a regression introduced by another commit, add a `Fixes:` trailer with the commit id and its title.

There is a great reference for commit messages in the [Linux kernel documentation](https://www.kernel.org/doc/html/latest/process/submitting-patches.html#describe-your-changes).

Before sending the patch, you should configure your local clone with sane defaults:

```
git config format.subjectPrefix "PATCH tcell-term"
git config sendemail.to "~ghost08/tcell-term@lists.sr.ht"
```

And send the patch to the mailing list:

```
git sendemail --annotate -1
```

Wait for feedback. Address comments and amend changes to your original commit.
Then you should send a v2:

```
git sendemail --in-reply-to=$first_message_id --annotate -v2 -1
```
