package errors

import (
	stderrors "errors"
	stdfmt "fmt"
	stdtesting "testing"
)

func noErrors(at, depth int) error {
	if at >= depth {
		return stderrors.New("no error")
	}

	return noErrors(at+1, depth)
}

func yesErrors(at, depth int) error {
	if at >= depth {
		return New("ye error")
	}

	return yesErrors(at+1, depth)
}

// GlobalE — экспортируемая глобальная переменная для хранения результата
// бенчмарков, чтобы компилятор не оптимизировал бенчмарк-функции.
var GlobalE any

func BenchmarkErrors(b *stdtesting.B) {
	type run struct {
		stack int
		std   bool
	}
	runs := []run{
		{10, false},
		{10, true},
		{100, false},
		{100, true},
		{1000, false},
		{1000, true},
	}
	for _, r := range runs {
		part := "pkg/errors"
		if r.std {
			part = "errors"
		}
		name := stdfmt.Sprintf("%s-stack-%d", part, r.stack)
		b.Run(name, func(b *stdtesting.B) {
			var err error
			f := yesErrors
			if r.std {
				f = noErrors
			}
			b.ReportAllocs()
			for range b.N {
				err = f(0, r.stack)
			}
			b.StopTimer()
			GlobalE = err
		})
	}
}

func BenchmarkStackFormatting(b *stdtesting.B) {
	type run struct {
		stack  int
		format string
	}
	runs := []run{
		{10, "%s"},
		{10, "%v"},
		{10, "%+v"},
		{30, "%s"},
		{30, "%v"},
		{30, "%+v"},
		{60, "%s"},
		{60, "%v"},
		{60, "%+v"},
	}

	var stackStr string
	for _, r := range runs {
		name := stdfmt.Sprintf("%s-stack-%d", r.format, r.stack)
		b.Run(name, func(b *stdtesting.B) {
			err := yesErrors(0, r.stack)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				stackStr = stdfmt.Sprintf(r.format, err)
			}
			b.StopTimer()
		})
	}

	for _, r := range runs {
		name := stdfmt.Sprintf("%s-stacktrace-%d", r.format, r.stack)
		b.Run(name, func(b *stdtesting.B) {
			err := yesErrors(0, r.stack)
			st := func() *fundamentalError {
				target := &fundamentalError{}
				_ = stderrors.As(err, &target)
				return target
			}().StackTrace()
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				stackStr = stdfmt.Sprintf(r.format, st)
			}
			b.StopTimer()
		})
	}
	GlobalE = stackStr
}
