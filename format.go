package errors

import (
	stderrors "errors"
	stdfmt "fmt"
)

func printDeep(s stdfmt.State, err error, st StackTracer) {
	if err == nil {
		return
	}

	stdfmt.Fprintln(s, err.Error())
	for err != nil {
		if x, ok := err.(StackTracer); ok {
			st = x
		}

		{
			var x interface{ Unwrap() error }
			var x1 interface{ Unwrap() []error }
			switch {
			case stderrors.As(err, &x):
				err = x.Unwrap()
				continue
			case stderrors.As(err, &x1):
				for i, e := range x1.Unwrap() {
					stdfmt.Fprintf(s, "%d. ", i+1)
					printDeep(s, e, st)
				}
			}
		}

		break
	}
	if st != nil {
		_, _ = st.StackTrace().WriteTo(s)
	}
}
