package main

import (
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/mjl-/bstore"
	"github.com/mjl-/sherpa"
)

func _checkf(err error, format string, args ...any) {
	if err == nil {
		return
	}

	msg := fmt.Sprintf(format, args...)

	if err == bstore.ErrAbsent {
		panic(&sherpa.Error{Code: "user:notFound", Message: msg + ": Not found"})
	}

	if errors.Is(err, bstore.ErrUnique) {
		err = fmt.Errorf("a value is not unique")
	} else if errors.Is(err, bstore.ErrReference) {
		err = fmt.Errorf("references to this object are still present in the database")
	} else if errors.Is(err, bstore.ErrZero) {
		err = fmt.Errorf("invalid empty value for a field")
	}

	m := msg
	if m != "" {
		m += ": "
	}
	m += err.Error()
	if config.PrintSherpaErrorStack {
		slog.Error("sherpa serverError", "err", m)
		debug.PrintStack()
	}
	if config.ShowSherpaErrors {
		m = msg + ": " + err.Error()
	} else {
		m = "An error occurred. Please try again later or contact us."
	}
	_serverError(m)
}

func _serverError(m string) {
	panic(&sherpa.Error{Code: "serverError", Message: m})
}

func _checkUserf(err error, format string, args ...any) {
	if err == nil {
		return
	}

	msg := fmt.Sprintf(format, args...)

	m := msg
	if m != "" {
		m += ": "
	}
	m += err.Error()
	if config.ShowSherpaErrors {
		m = msg + ": " + err.Error()
	} else {
		m = "An error occurred. Please try again later or contact us."
	}
	_userError(m)
}

func _userError(m string) {
	panic(&sherpa.Error{Code: "user:error", Message: m})
}

func sherpaCatch(fn func()) (rerr error) {
	defer func() {
		x := recover()
		if x == nil {
			return
		}
		err, ok := x.(*sherpa.Error)
		if !ok {
			panic(x)
		}
		rerr = err
	}()
	fn()
	return nil
}
