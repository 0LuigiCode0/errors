package errors_test

import (
	stdfmt "fmt"
	stdio "io"
	stdstrings "strings"

	"github.com/0LuigiCode0/errors"
)

// printFnNames печатает только имена функций из расширенного вывода,
// обрезая часть «file:line» у строк кадра «func file:line» и пропуская
// пустые строки, чтобы пример не зависел от абсолютных путей к модулю.
// Строка считается кадром, если хвост после последнего пробела имеет вид
// «file:line» с числовой строкой (сообщения об ошибках так не оканчиваются).
func printFnNames(s string) {
	for ln := range stdstrings.SplitSeq(s, "\n") {
		if ln == "" {
			continue
		}
		if idx := stdstrings.LastIndex(ln, " "); idx > 0 && isFrameTail(ln[idx+1:]) {
			ln = ln[:idx]
		}
		stdfmt.Println(ln)
	}
}

// isFrameTail сообщает, является ли tail хвостом кадра «file:line»:
// содержит «:», после которого идут только цифры.
func isFrameTail(tail string) bool {
	colon := stdstrings.LastIndex(tail, ":")
	if colon < 0 || colon == len(tail)-1 {
		return false
	}
	for _, r := range tail[colon+1:] {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

func init() {
	errors.CutTail = 3
}

func ExampleNew() {
	err := errors.New("whoops")
	stdfmt.Println(err)

	// Output: whoops
}

func ExampleNew_printf() {
	err := errors.New("whoops")
	printFnNames(stdfmt.Sprintf("%+v", err))

	// Output:
	// whoops
	// github.com/0LuigiCode0/errors_test.ExampleNew_printf
	// testing.runExample
	// testing.runExamples
	// testing.(*M).Run
}

func ExampleWithMessage() {
	cause := errors.New("whoops")
	err := errors.WithMessage(cause, "oh noes")
	stdfmt.Println(err)

	// Output: oh noes: whoops
}

func ExampleWithMessage_global() {
	stdfmt.Println(errors.ErrExm1)

	// Output: foo
}

//go:noinline
func returnWithStack(err error) error { return errors.WithStack(err) }

func ExampleWithStack() {
	stdfmt.Println(returnWithStack(errors.New("whoops")))

	// Output: whoops
}

func ExampleWithStack_printf() {
	printFnNames(stdfmt.Sprintf("%+v", returnWithStack(errors.New("whoops"))))

	// Output:
	// whoops
	// github.com/0LuigiCode0/errors_test.ExampleWithStack_printf
	// testing.runExample
	// testing.runExamples
	// testing.(*M).Run
}

func ExampleWrap() {
	cause := errors.New("whoops")
	err := errors.Wrap(cause, "oh noes")
	stdfmt.Println(err)

	// Output: oh noes: whoops
}

func fn() error {
	e1 := errors.New("error")
	e2 := errors.Wrap(e1, "inner")
	e3 := errors.Wrap(e2, "middle")

	return errors.Wrap(e3, "outer")
}

func ExampleCause() {
	err := fn()
	stdfmt.Println(err)
	stdfmt.Println(errors.Cause(err))

	// Output: outer: middle: inner: error
	// error
}

func ExampleWrap_extended() {
	err := fn()
	printFnNames(stdfmt.Sprintf("%+v", err))

	// Output:
	// outer: middle: inner: error
	// github.com/0LuigiCode0/errors_test.fn
	// github.com/0LuigiCode0/errors_test.ExampleWrap_extended
	// testing.runExample
	// testing.runExamples
	// testing.(*M).Run
}

func ExampleWrap_extended_global() {
	err := errors.Wrap(errors.ErrExm1, "outer")
	printFnNames(stdfmt.Sprintf("%+v", err))

	// Output:
	// outer: foo
	// github.com/0LuigiCode0/errors_test.ExampleWrap_extended_global
	// testing.runExample
	// testing.runExamples
	// testing.(*M).Run
}

func ExampleWrapf() {
	cause := errors.New("whoops")
	err := errors.Wrapf(cause, "oh noes #%d", 2)
	stdfmt.Println(err)

	// Output: oh noes #2: whoops
}

func ExampleErrorf_extended() {
	err := errors.Newf("whoops: %s", "foo")
	printFnNames(stdfmt.Sprintf("%+v", err))

	// Output:
	// whoops: foo
	// github.com/0LuigiCode0/errors_test.ExampleErrorf_extended
	// testing.runExample
	// testing.runExamples
	// testing.(*M).Run
}

func Example_stackTrace() {
	err, ok := errors.Cause(fn()).(errors.StackTracer)
	if !ok {
		panic("oops, err does not implement stackTracer")
	}

	st := err.StackTrace()
	printFnNames(stdfmt.Sprintf("%+v", st)) // top two frames

	// Output:
	// github.com/0LuigiCode0/errors_test.fn
	// github.com/0LuigiCode0/errors_test.Example_stackTrace
	// testing.runExample
	// testing.runExamples
	// testing.(*M).Run
}

func ExampleCause_printf() {
	err := errors.Wrap(func() error {
		return func() error {
			return errors.New("hello world")
		}()
	}(), "failed")

	stdfmt.Printf("%v", err)

	// Output: failed: hello world
}

func ExampleWrapMulti() {
	err1 := errors.New("whoops")
	err2 := errors.New("oops")
	printFnNames(stdfmt.Sprintf("%+v", errors.Wrap(errors.WrapMulti([]error{stdio.EOF, err1, err2}, "pops"), "pops")))

	// Output:
	// pops: pops: EOF
	// whoops
	// oops
	// 1. EOF
	// github.com/0LuigiCode0/errors_test.ExampleWrapMulti
	// testing.runExample
	// testing.runExamples
	// testing.(*M).Run
	// 2. whoops
	// github.com/0LuigiCode0/errors_test.ExampleWrapMulti
	// testing.runExample
	// testing.runExamples
	// testing.(*M).Run
	// 3. oops
	// github.com/0LuigiCode0/errors_test.ExampleWrapMulti
	// testing.runExample
	// testing.runExamples
	// testing.(*M).Run
	// github.com/0LuigiCode0/errors_test.ExampleWrapMulti
	// testing.runExample
	// testing.runExamples
	// testing.(*M).Run
}
