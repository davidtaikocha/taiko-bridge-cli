package clierr

import "fmt"

type Error struct {
	Code int
	Err  error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return fmt.Sprintf("exit code %d", e.Code)
	}
	return e.Err.Error()
}

func Wrap(code int, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Code: code, Err: err}
}

func Code(err error, fallback int) int {
	if err == nil {
		return 0
	}
	e, ok := err.(*Error)
	if !ok {
		return fallback
	}
	return e.Code
}
