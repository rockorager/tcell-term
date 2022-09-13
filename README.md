# tcell-term

A virtual terminal widget for [tcell](https://github.com/gdamore/tcell/)

tcell-term implements the native tcell Widget interface.

```go
screen := tcell.NewScreen()
term := tcellterm.New()
// Create a view. A screen is also a valid view
view := views.NewViewport(screen, 0, 0, -1, -1)

// Set the view. This must be set before calling Draw in your event
// handler
term.SetView(view)

// Call watch with your model. It should HandleEvent(ev tcell.Event)
term.Watch(myWidgetEventWatcher)

cmd := exec.Command(os.Getenv("SHELL"))

go func() {
	term.Run(cmd)
}()
```

For general discussion or patches, use the [mailing list](https://lists.sr.ht/~rockorager/tcell-term): [~rockorager/tcell-term@lists.sr.ht](mailto:~rockorager/tcell-term@lists.sr.ht).

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
git config sendemail.to "~rockorager/tcell-term@lists.sr.ht"
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
