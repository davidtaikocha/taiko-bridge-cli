package clierr

import "fmt"

// Error is a typed CLI error carrying an exit code and wrapped cause.
type Error struct {
	// Code is the process exit code to return.
	Code int
	// Err is the wrapped underlying error.
	Err error
}

// Error renders the wrapped error text for CLI output.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return fmt.Sprintf("exit code %d", e.Code)
	}
	return e.Err.Error()
}

// Wrap attaches a CLI exit code to an error.
func Wrap(code int, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Code: code, Err: err}
}

// Code extracts a CLI exit code from err, or returns fallback.
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
