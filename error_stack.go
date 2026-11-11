package errors

import (
	"encoding/json"
	stdfmt "fmt"
)

// withStackError — обёртка, добавляющая к ошибке трассировку стека и код.
type withStackError struct {
	Stack
	error

	code int
}

func (e *withStackError) Code() int                    { return e.code }
func (e *withStackError) Error() string                { return e.String() }
func (e *withStackError) String() string               { return e.error.Error() }
func (e *withStackError) MarshalJSON() ([]byte, error) { return json.Marshal(e.Error()) }

func (e *withStackError) Format(s stdfmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			printDeep(s, e, e)
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
func (e *withStackError) Cause() error { return e.error }

// Unwrap обеспечивает совместимость с цепочками ошибок Go 1.13.
func (e *withStackError) Unwrap() error { return e.error }
