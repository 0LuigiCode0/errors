package errors

import (
	stderrors "errors"
	stdfmt "fmt"
)

// NaNCode — код ошибки, у которой код не задан.
const NaNCode = -1

// NewCode возвращает ошибку с заданным кодом и сообщением.
// NewCode также записывает трассировку стека в точке вызова.
func NewCode(code int, message string) error {
	return &fundamentalError{
		msg:   message,
		Stack: callers(),
		code:  code,
	}
}

// NewCodef — версия NewCode, в которой сообщение форматируется по
// спецификатору формата.
func NewCodef(code int, format string, args ...any) error {
	return &fundamentalError{
		msg:   stdfmt.Sprintf(format, args...),
		Stack: callers(),
		code:  code,
	}
}

// WithStackCode — версия WithStack, которая дополнительно помечает err
// заданным кодом.
// Если err равен nil, WithStackCode возвращает nil.
func WithStackCode(code int, err error) error {
	if err == nil {
		return nil
	}

	return &withStackError{
		error: err,
		Stack: callers(),
		code:  code,
	}
}

// WrapCode — версия Wrap, которая дополнительно помечает обёртку заданным
// кодом.
// Если err равен nil, WrapCode возвращает nil.
func WrapCode(code int, err error, message string) error {
	if err == nil {
		return nil
	}

	return &withStackError{
		error: &withMessageError{
			error: err,
			msg:   message,
			code:  NaNCode,
		},
		Stack: callers(),
		code:  code,
	}
}

// WrapCodef — версия Wrapf, которая дополнительно помечает обёртку
// заданным кодом.
// Если err равен nil, WrapCodef возвращает nil.
func WrapCodef(code int, err error, format string, args ...any) error {
	if err == nil {
		return nil
	}

	return &withStackError{
		error: &withMessageError{
			error: err,
			msg:   stdfmt.Sprintf(format, args...),
			code:  NaNCode,
		},
		Stack: callers(),
		code:  code,
	}
}

// WrapMultiCode — версия WrapMulti, которая дополнительно помечает обёртку
// заданным кодом.
// Ошибки, равные nil, пропускаются. Если не-nil ошибок нет, WrapMultiCode
// возвращает nil.
func WrapMultiCode(code int, errs []error, message string) error {
	errs = filterNil(errs)
	if len(errs) == 0 {
		return nil
	}

	return &withStackError{
		error: &withMessageError{
			error: stderrors.Join(errs...),
			msg:   message,
			code:  NaNCode,
		},
		Stack: callers(),
		code:  code,
	}
}

// WrapMultiCodef — версия WrapMultiCode, в которой сообщение форматируется
// по спецификатору формата.
func WrapMultiCodef(code int, errs []error, format string, args ...any) error {
	errs = filterNil(errs)
	if len(errs) == 0 {
		return nil
	}

	return &withStackError{
		error: &withMessageError{
			error: stderrors.Join(errs...),
			msg:   stdfmt.Sprintf(format, args...),
			code:  NaNCode,
		},
		Stack: callers(),
		code:  code,
	}
}

// WithMessageCode — версия WithMessage, которая дополнительно помечает
// обёртку заданным кодом.
// Если err равен nil, WithMessageCode создает ошибку с текстом.
func WithMessageCode(code int, err error, message string) error {
	return &withMessageError{
		error: err,
		msg:   message,
		code:  code,
	}
}

// WithMessageCodef — версия WithMessagef, которая дополнительно помечает
// обёртку заданным кодом.
// Если err равен nil, WithMessageCodef создает ошибку с текстом.
func WithMessageCodef(code int, err error, format string, args ...any) error {
	return &withMessageError{
		error: err,
		msg:   stdfmt.Sprintf(format, args...),
		code:  code,
	}
}
