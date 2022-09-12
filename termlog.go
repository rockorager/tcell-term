package tcellterm

type logger struct {
	fn func(string, ...interface{})
}

var tlog logger

func SetLogger(l func(string, ...interface{})) {
	tlog.fn = l
}

func (l *logger) Printf(s string, args ...interface{}) {
	if l.fn == nil {
		return
	}
	l.fn(s, args...)
}
