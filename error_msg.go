package errors

import (
	"encoding/json"
	stdfmt "fmt"
)

// withMessageError — обёртка, добавляющая к ошибке сообщение и код.
type withMessageError struct {
	error

	msg  string
	code int
}

func (e *withMessageError) Code() int     { return e.code }
func (e *withMessageError) Error() string { return e.String() }
func (e *withMessageError) String() string {
	if e.error == nil {
		return e.msg
	}

	return stdfmt.Sprintf("%s: %v", e.msg, e.error)
}
func (e *withMessageError) MarshalJSON() ([]byte, error) { return json.Marshal(e.Error()) }

func (e *withMessageError) Format(s stdfmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			printDeep(s, e, nil)
			return
		}

		fallthrough
	case 's':
		stdfmt.Fprint(s, e.String())
	case 'q':
		stdfmt.Fprintf(s, "%q", e.String())
	default:
		stdfmt.Fprintf(s, string(verb), *e)
	}
}

// Cause возвращает обёрнутую ошибку.
func (e *withMessageError) Cause() error { return e.error }

// Unwrap обеспечивает совместимость с цепочками ошибок Go 1.13.
func (e *withMessageError) Unwrap() error { return e.error }
