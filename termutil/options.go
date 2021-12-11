package termutil

type Option func(t *Terminal)

func WithTheme(theme *Theme) Option {
	return func(t *Terminal) {
		t.theme = theme
	}
}

func WithWindowManipulator(m WindowManipulator) Option {
	return func(t *Terminal) {
		t.windowManipulator = m
	}
}
