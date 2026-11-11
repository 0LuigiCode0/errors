package errors

import (
	"encoding/json"
	stdfmt "fmt"
)

type fundamentalError struct {
	Stack

	msg  string
	code int
}

func (e *fundamentalError) Code() int                    { return e.code }
func (e *fundamentalError) Error() string                { return e.String() }
func (e *fundamentalError) String() string               { return e.msg }
func (e *fundamentalError) MarshalJSON() ([]byte, error) { return json.Marshal(e.Error()) }

func (e *fundamentalError) Format(s stdfmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			stdfmt.Fprintln(s, e.String())
			_, _ = e.WriteTo(s)
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
