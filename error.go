// Package errors предоставляет примитивы для обработки ошибок: добавление контекста, трассировку стека, числовые коды ошибок и мульти-ошибки.
//
// Традиционная обработка ошибок в Go строится на идоме
//
//	if err != nil {
//	        return err
//	}
//
// который при рекурсивном применении вверх по стеку вызовов приводит к
// сообщениям об ошибках без контекста и отладочной информации. Пакет errors
// позволяет добавлять контекст на путь ошибки, не разрушая исходное
// значение ошибки.
//
// # Добавление контекста к ошибке
//
// Функция errors.Wrap возвращает новую ошибку, которая добавляет контекст к
// исходной: записывает трассировку стека в точке вызова Wrap вместе с
// переданным сообщением. Например
//
//	_, err := io.ReadAll(r)
//	if err != nil {
//	        return errors.Wrap(err, "read failed")
//	}
//
// Если нужен более тонкий контроль, функции errors.WithStack и
// errors.WithMessage разлагают errors.Wrap на составляющие операции:
// пометку ошибки трассировкой стека и сообщением соответственно.
//
// # Получение причины ошибки
//
// Использование errors.Wrap строит стек ошибок, добавляя контекст к
// предыдущей ошибке. Иногда требуется развернуть цепочку, чтобы получить
// исходную ошибку. Любое значение ошибки, реализующее интерфейс
//
//	type causer interface {
//	        Cause() error
//	}
//
// можно исследовать с помощью errors.Cause. errors.Cause рекурсивно
// возвращает верхнюю ошибку, не реализующую causer, которая считается
// первопричиной. Например:
//
//	switch err := errors.Cause(err).(type) {
//	case *MyError:
//	        // обработать конкретно
//	default:
//	        // неизвестная ошибка
//	}
//
// Хотя интерфейс causer не экспортируется пакетом, он считается частью его
// стабильного публичного интерфейса.
//
// # Форматированный вывод ошибок
//
// Все значения ошибок, возвращаемые этим пакетом, реализуют fmt.Formatter и
// могут форматироваться пакетом fmt. Поддерживаются следующие вербы:
//
//	%s    выводит ошибку. Если у ошибки есть Cause, она
//	      выводится рекурсивно.
//	%v    аналогично %s
//	%+v   расширенный формат. Каждый кадр трассировки стека
//	      ошибки выводится в деталях.
//
// # Получение трассировки стека ошибки или обёртки
//
// New, Newf, WithStack и все функции Wrap* записывают трассировку стека в
// точке вызова. Доступ к ней можно получить через интерфейс StackTracer:
//
//	type StackTracer interface {
//	        StackTrace() Stack
//	}
//
// Stack — срез точек вызова:
//
//	type Stack []FuncPtr
//
// Каждый FuncPtr при выводе печатается одной строкой «func file:line».
// Например:
//
//	if st, ok := err.(StackTracer); ok {
//	        fmt.Println(st.StackTrace())
//	}
//
// Интерфейс StackTracer экспортируется и считается частью стабильного
// публичного интерфейса пакета.
package errors

import (
	stderrors "errors"
	stdfmt "fmt"
	stdslices "slices"
)

// Is сообщает, совпадает ли какая-либо ошибка в цепочке err с target.
//
// Цепочка состоит из самой err и последовательности ошибок, полученных
// повторными вызовами Unwrap.
//
// Ошибка считается совпавшей, если она равна target или реализует метод
// Is(error) bool, такой что Is(target) возвращает true.
func Is(err, target error) bool { return stderrors.Is(err, target) }

// Iss — вариативная версия Is: сообщает, совпадает ли какая-либо ошибка в
// цепочке err с одним из targets.
//
// Цепочка состоит из самой err, всех ошибок, достижимых повторными вызовами
// Unwrap() error, и каждой ошибки, достижимой через Unwrap() []error
// (мульти-ошибки, например из errors.Join).
//
// Ошибка считается совпавшей, если она равна одному из targets или
// реализует метод Is(error) bool, такой что Is(target) возвращает true для
// одного из targets.
//
// Если err равна nil, Iss возвращает false.
func Iss(err error, targets ...error) bool {
	if err == nil {
		return false
	}
	for {
		x, ok := err.(interface{ Is(err error) bool })
		for _, target := range targets {
			if ok && x.Is(target) || err == target {
				return true
			}
		}
		{
			var x interface{ Unwrap() error }
			var x1 interface{ Unwrap() []error }
			switch {
			case stderrors.As(err, &x):
				err = x.Unwrap()
				if err == nil {
					return false
				}
			case stderrors.As(err, &x1):
				for _, err := range x1.Unwrap() {
					if Iss(err, targets...) {
						return true
					}
				}

				return false
			default:
				return false
			}
		}
	}
}

// As находит первую ошибку в цепочке err, совпадающую с target, и при
// совпадении записывает в target это значение ошибки, возвращая true.
//
// Цепочка состоит из самой err и последовательности ошибок, полученных
// повторными вызовами Unwrap.
//
// Ошибка совпадает с target, если конкретное значение ошибки присваивается
// значению, на которое указывает target, или если у ошибки есть метод
// As(interface{}) bool, такой что As(target) возвращает true. Во втором
// случае метод As отвечает за запись в target.
//
// As паникует, если target не является не-nil указателем на тип,
// реализующий error, или на любой интерфейс. As возвращает false, если err
// равна nil.
func As(err error, target any) bool { return stderrors.As(err, target) }

// Unwrap возвращает результат вызова метода Unwrap у err, если тип err
// содержит метод Unwrap, возвращающий error.
// В противном случае Unwrap возвращает nil.
func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}

// IsCode сообщает, совпадает ли код какой-либо ошибки в цепочке err с одним
// из codes.
//
// Цепочка состоит из самой err, всех ошибок, достижимых повторными вызовами
// Unwrap() error, и каждой ошибки, достижимой через Unwrap() []error
// (мульти-ошибки, например из errors.Join).
//
// Ошибка, не реализующая метод Code() int, кода не имеет.
//
// Если err равна nil или codes пуста, IsCode возвращает false.
func IsCode(err error, codes ...int) bool {
	if err == nil {
		return false
	}
	for {
		if x, ok := err.(interface{ Code() int }); ok {
			errCode := x.Code()
			if stdslices.Contains(codes, errCode) {
				return true
			}
		}
		{
			var x interface{ Unwrap() error }
			var x1 interface{ Unwrap() []error }
			switch {
			case stderrors.As(err, &x):
				err = x.Unwrap()
				if err == nil {
					return false
				}
			case stderrors.As(err, &x1):
				for _, err := range x1.Unwrap() {
					if IsCode(err, codes...) {
						return true
					}
				}

				return false
			default:
				return false
			}
		}
	}
}

// Cause возвращает первопричину ошибки, если это возможно.
// У значения ошибки есть причина, если оно реализует интерфейс
//
//	type causer interface {
//	       Cause() error
//	}
//
// Если ошибка не реализует Cause, возвращается исходная ошибка.
// Если ошибка равна nil, возвращается nil без дальнейшего исследования.
func Cause(err error) error {
	type causer interface {
		Cause() error
	}

	for err != nil {
		cause, ok := err.(causer)
		if !ok {
			break
		}
		err = cause.Cause()
	}

	return err
}

// New возвращает ошибку с переданным сообщением.
// New также записывает трассировку стека в точке вызова.
func New(message string) error {
	return &fundamentalError{
		msg:   message,
		Stack: callers(),
		code:  NaNCode,
	}
}

// Newf форматирует строку по спецификатору формата и возвращает её как
// значение, удовлетворяющее error.
// Newf также записывает трассировку стека в точке вызова.
func Newf(format string, args ...any) error {
	return &fundamentalError{
		msg:   stdfmt.Sprintf(format, args...),
		Stack: callers(),
		code:  NaNCode,
	}
}

// WithStack помечает err трассировкой стека в точке вызова WithStack.
// Если err равна nil, WithStack возвращает nil.
func WithStack(err error) error {
	if err == nil {
		return nil
	}

	return &withStackError{
		error: err,
		Stack: callers(),
		code:  NaNCode,
	}
}

// Wrap возвращает ошибку, помечающую err трассировкой стека в точке вызова
// Wrap и переданным сообщением.
// Если err равна nil, Wrap возвращает nil.
func Wrap(err error, message string) error {
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
		code:  NaNCode,
	}
}

// Wrapf возвращает ошибку, помечающую err трассировкой стека в точке вызова
// Wrapf и сообщением, форматированным по спецификатору формата.
// Если err равна nil, Wrapf возвращает nil.
func Wrapf(err error, format string, args ...any) error {
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
		code:  NaNCode,
	}
}

// WrapMulti возвращает ошибку, помечающую все не-nil ошибки из errs,
// объединённые стандартными errors.Join, трассировкой стека в точке вызова
// WrapMulti и переданным сообщением.
// Ошибки равные nil пропускаются. Если не-nil ошибок нет, WrapMulti
// возвращает nil.
func WrapMulti(errs []error, message string) error {
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
		code:  NaNCode,
	}
}

// WrapMultif — версия WrapMulti, где сообщение форматируется по
// спецификатору формата.
func WrapMultif(errs []error, format string, args ...any) error {
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
		code:  NaNCode,
	}
}

// filterNil удаляет nil-ошибки из среза.
func filterNil(errs []error) []error {
	out := make([]error, 0, len(errs))
	for _, e := range errs {
		if e != nil {
			out = append(out, e)
		}
	}

	return out
}

// WithMessage помечает err новым сообщением.
// Если err равна nil, WithMessage создает ошибку с текстом.
func WithMessage(err error, message string) error {
	return &withMessageError{
		error: err,
		msg:   message,
		code:  NaNCode,
	}
}

// WithMessagef помечает err сообщением, форматированным по спецификатору
// формата.
// Если err равна nil, WithMessagef создает ошибку с текстом.
func WithMessagef(err error, format string, args ...any) error {
	return &withMessageError{
		error: err,
		msg:   stdfmt.Sprintf(format, args...),
		code:  NaNCode,
	}
}
