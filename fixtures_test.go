package errors

import (
	stderrors "errors"
	stdio "io"
)

// Фикстуры для формат- и стек-тестов: каждый способ создания ошибки
// вынесен в именованную функцию, чтобы в ожидаемых стеках проверялось
// имя функции и файл, а не номер строки. Все фикстуры помечены
// //go:noinline, чтобы инлайнер не подменял фрейм фикстуры на
// фрейм вызывающего теста.
const (
	testModule    = `github\.com/0LuigiCode0/errors`
	fixtureFile   = "fixtures_test.go"
	stackTestFile = "stack_test.go"
)

var ErrExm1 = WithMessage(nil, "foo")

var initpc = caller()

//go:noinline
func fixtureNew() error { return New("error") }

//go:noinline
func fixtureNewf() error { return Newf("%s", "error") }

//go:noinline
func fixtureWrappedNew(message string) error { return New(message) }

//go:noinline
func fixtureWrapNew() error { return Wrap(New("error"), "error2") }

//go:noinline
func fixtureWrapEOF() error { return Wrap(stdio.EOF, "error") }

//go:noinline
func fixtureDoubleWrap() error { return Wrap(Wrap(stdio.EOF, "error1"), "error2") }

//go:noinline
func fixtureWrapContext() error { return Wrap(New("error with space"), "context") }

//go:noinline
func fixtureWrapfEOF() error { return Wrapf(stdio.EOF, "error%d", 2) }

//go:noinline
func fixtureWrapfNew() error { return Wrapf(New("error"), "error%d", 2) }

//go:noinline
func fixtureWithStackEOF() error { return WithStack(stdio.EOF) }

//go:noinline
func fixtureWithStackNew() error { return WithStack(New("error")) }

//go:noinline
func fixtureDoubleWithStack() error { return WithStack(WithStack(stdio.EOF)) }

//go:noinline
func fixtureWithStackWrapfEOF() error { return WithStack(WithStack(Wrapf(stdio.EOF, "message"))) }

//go:noinline
func fixtureWithStackNewf() error { return WithStack(Newf("error%d", 1)) }

//go:noinline
func fixtureWithMessageNew() error { return WithMessage(New("error"), "error2") }

//go:noinline
func fixtureWithMessageEOF() error { return WithMessage(stdio.EOF, "addition1") }

//go:noinline
func fixtureDoubleWithMessage() error {
	return WithMessage(WithMessage(stdio.EOF, "addition1"), "addition2")
}

//go:noinline
func fixtureWrapOfWithMessage() error { return Wrap(WithMessage(stdio.EOF, "error1"), "error2") }

//go:noinline
func fixtureWithMessageNewf() error { return WithMessage(Newf("error%d", 1), "error2") }

//go:noinline
func fixtureWithMessageWithStack() error { return WithMessage(WithStack(stdio.EOF), "error") }

//go:noinline
func fixtureWithMessageWrapWithStack() error {
	return WithMessage(Wrap(WithStack(stdio.EOF), "inside-error"), "outside-error")
}

//go:noinline
func fixtureGenericNew() error { return New("new-error") }

//go:noinline
func fixtureGenericNewf() error { return Newf("errorf-error") }

//go:noinline
func fixtureGenericStdlibNew() error { return stderrors.New("errors-new-error") }

//go:noinline
func fixtureWrapWithMessage(err error) error { return WithMessage(err, "with-message") }

//go:noinline
func fixtureWrapWithStack(err error) error { return WithStack(err) }

//go:noinline
func fixtureWrapWrap(err error) error { return Wrap(err, "wrap-error") }

//go:noinline
func fixtureWrapWrapf(err error) error { return Wrapf(err, "wrapf-error%d", 1) }

//go:noinline
func fixtureStackTraceNew() error { return New("ooh") }

//go:noinline
func fixtureStackTraceWrap() error { return Wrap(New("ooh"), "ahh") }

//go:noinline
func fixtureStackTraceNested() error { return fixtureStackTraceNestedInner() }

//go:noinline
func fixtureStackTraceNestedInner() error {
	return Newf("hello %s", "world: "+"ooh")
}
