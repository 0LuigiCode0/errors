//go:build go1.13

package errors

import (
	stderrors "errors"
	stdfmt "fmt"
	stdio "io"
	stdreflect "reflect"
	stdtesting "testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorChainCompat(t *stdtesting.T) {
	err := stderrors.New("error that gets wrapped")
	wrapped := Wrap(err, "wrapped up")
	if !stderrors.Is(wrapped, err) {
		t.Errorf("Wrap does not support Go 1.13 error chains")
	}
}

func TestIs(t *stdtesting.T) {
	err := New("test")

	type args struct {
		err    error
		target error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "with stack",
			args: args{
				err:    WithStack(err),
				target: err,
			},
			want: true,
		},
		{
			name: "with message",
			args: args{
				err:    WithMessage(err, "test"),
				target: err,
			},
			want: true,
		},
		{
			name: "with message format",
			args: args{
				err:    WithMessagef(err, "%s", "test"),
				target: err,
			},
			want: true,
		},
		{
			name: "std errors compatibility",
			args: args{
				err:    stdfmt.Errorf("wrap it: %w", err),
				target: err,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *stdtesting.T) {
			if got := Is(tt.args.err, tt.args.target); got != tt.want {
				t.Errorf("Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

type customError struct {
	msg string
}

func (c customError) Error() string { return c.msg }

func TestAs(t *stdtesting.T) {
	err := customError{msg: "test message"}

	type args struct {
		err    error
		target any
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "with stack",
			args: args{
				err:    WithStack(err),
				target: new(customError),
			},
			want: true,
		},
		{
			name: "with message",
			args: args{
				err:    WithMessage(err, "test"),
				target: new(customError),
			},
			want: true,
		},
		{
			name: "with message format",
			args: args{
				err:    WithMessagef(err, "%s", "test"),
				target: new(customError),
			},
			want: true,
		},
		{
			name: "std errors compatibility",
			args: args{
				err:    stdfmt.Errorf("wrap it: %w", err),
				target: new(customError),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *stdtesting.T) {
			if got := As(tt.args.err, tt.args.target); got != tt.want {
				t.Errorf("As() = %v, want %v", got, tt.want)
			}

			ce := tt.args.target.(*customError)
			if !stdreflect.DeepEqual(err, *ce) {
				t.Errorf("set target error failed, target error is %v", *ce)
			}
		})
	}
}

func TestUnwrap(t *stdtesting.T) {
	err := New("test")

	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{
			name: "with stack",
			args: args{err: WithStack(err)},
			want: err,
		},
		{
			name: "with message",
			args: args{err: WithMessage(err, "test")},
			want: err,
		},
		{
			name: "with message format",
			args: args{err: WithMessagef(err, "%s", "test")},
			want: err,
		},
		{
			name: "std errors compatibility",
			args: args{err: stdfmt.Errorf("wrap: %w", err)},
			want: err,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *stdtesting.T) {
			if err := Unwrap(tt.args.err); !assert.Equal(t, tt.want, err) {
				t.Errorf("Unwrap() error = %v, want %v", err, tt.want)
			}
		})
	}
}

// codedError — минимальная ошибка с кодом.
type codedError struct {
	msg  string
	code int
}

func (e *codedError) Error() string { return e.msg }
func (e *codedError) Code() int     { return e.code }

// unwrappingError — минимальная обёртка цепочки в одну ошибку.
type unwrappingError struct{ err error }

func (e unwrappingError) Error() string { return e.err.Error() }
func (e unwrappingError) Unwrap() error { return e.err }

// customIsError совпадает только со своим target через метод Is.
type customIsError struct{ target error }

func (e *customIsError) Error() string        { return "custom" }
func (e *customIsError) Is(target error) bool { return e.target == target }

func TestIsCode(t *stdtesting.T) {
	tests := []struct {
		name  string
		err   error
		codes []int
		want  bool
	}{
		{
			name:  "nil error",
			err:   nil,
			codes: []int{1},
			want:  false,
		},
		{
			name:  "direct match",
			err:   NewCode(1, "message"),
			codes: []int{1},
			want:  true,
		},
		{
			name:  "direct miss",
			err:   NewCode(1, "message"),
			codes: []int{2, 3},
			want:  false,
		},
		{
			name:  "match in codes list",
			err:   NewCode(1, "message"),
			codes: []int{2, 3, 1},
			want:  true,
		},
		{
			name:  "no codes",
			err:   NewCode(1, "message"),
			codes: nil,
			want:  false,
		},
		{
			name:  "new has nan code",
			err:   New("message"),
			codes: []int{NaNCode},
			want:  true,
		},
		{
			name:  "new codef",
			err:   NewCodef(2, "formatted %s", "message"),
			codes: []int{2},
			want:  true,
		},
		{
			name:  "with stack code outer",
			err:   WithStackCode(4, NewCode(2, "message")),
			codes: []int{4},
			want:  true,
		},
		{
			name:  "with stack code inner",
			err:   WithStackCode(4, NewCode(2, "message")),
			codes: []int{2},
			want:  true,
		},
		{
			name:  "with stack code miss",
			err:   WithStackCode(4, NewCode(2, "message")),
			codes: []int{5},
			want:  false,
		},
		{
			name:  "with message code outer",
			err:   WithMessageCode(7, NewCode(3, "message"), "context"),
			codes: []int{7},
			want:  true,
		},
		{
			name:  "with message code inner",
			err:   WithMessageCode(7, NewCode(3, "message"), "context"),
			codes: []int{3},
			want:  true,
		},
		{
			name:  "with message code miss",
			err:   WithMessageCode(7, NewCode(3, "message"), "context"),
			codes: []int{9},
			want:  false,
		},
		{
			name:  "with message codef",
			err:   WithMessageCodef(6, NewCode(3, "message"), "context %s", "x"),
			codes: []int{6},
			want:  true,
		},
		{
			name:  "wrap code outer",
			err:   WrapCode(5, NewCode(1, "message"), "context"),
			codes: []int{5},
			want:  true,
		},
		{
			name:  "wrap code inner",
			err:   WrapCode(5, NewCode(1, "message"), "context"),
			codes: []int{1},
			want:  true,
		},
		{
			name:  "wrap code miss",
			err:   WrapCode(5, NewCode(1, "message"), "context"),
			codes: []int{2},
			want:  false,
		},
		{
			name:  "wrap codef",
			err:   WrapCodef(8, NewCode(1, "message"), "context %s", "x"),
			codes: []int{8},
			want:  true,
		},
		{
			name:  "wrap without code keeps inner code",
			err:   Wrap(NewCode(1, "message"), "context"),
			codes: []int{1},
			want:  true,
		},
		{
			name:  "wrap without code has nan outside",
			err:   Wrap(NewCode(1, "message"), "context"),
			codes: []int{NaNCode},
			want:  true,
		},
		{
			name:  "deep chain finds code",
			err:   WrapCode(5, WithStackCode(4, WithMessageCode(7, NewCode(3, "message"), "m")), "w"),
			codes: []int{3},
			want:  true,
		},
		{
			name:  "wrap of nil is nil",
			err:   WrapCode(1, nil, "context"),
			codes: []int{1},
			want:  false,
		},
		{
			name:  "stdlib error has no code",
			err:   stdio.EOF,
			codes: []int{1},
			want:  false,
		},
		{
			name:  "multi error match",
			err:   stderrors.Join(NewCode(9, "a"), NewCode(10, "b")),
			codes: []int{10},
			want:  true,
		},
		{
			name:  "multi error match first",
			err:   stderrors.Join(NewCode(9, "a"), NewCode(10, "b")),
			codes: []int{9},
			want:  true,
		},
		{
			name:  "multi error miss",
			err:   stderrors.Join(NewCode(9, "a"), NewCode(10, "b")),
			codes: []int{11},
			want:  false,
		},
		{
			name:  "multi error with nil",
			err:   stderrors.Join(nil, NewCode(12, "a")),
			codes: []int{12},
			want:  true,
		},
		{
			name:  "custom coded type",
			err:   &codedError{msg: "c", code: 8},
			codes: []int{8},
			want:  true,
		},
		{
			name:  "custom coded type miss",
			err:   &codedError{msg: "c", code: 8},
			codes: []int{1},
			want:  false,
		},
		{
			name:  "unwrapping to coded",
			err:   unwrappingError{err: NewCode(3, "message")},
			codes: []int{3},
			want:  true,
		},
		{
			name:  "unwrapping chain to coded",
			err:   unwrappingError{err: unwrappingError{err: &codedError{msg: "c", code: 15}}},
			codes: []int{15},
			want:  true,
		},
		{
			name:  "wrapped unwrapping to coded",
			err:   Wrap(unwrappingError{err: &codedError{msg: "c", code: 21}}, "context"),
			codes: []int{21},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *stdtesting.T) {
			if got := IsCode(tt.err, tt.codes...); got != tt.want {
				t.Errorf("IsCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIss(t *stdtesting.T) {
	base := New("base")

	tests := []struct {
		name    string
		err     error
		targets []error
		want    bool
	}{
		{
			name:    "nil error",
			err:     nil,
			targets: []error{base},
			want:    false,
		},
		{
			name:    "direct equal",
			err:     base,
			targets: []error{base},
			want:    true,
		},
		{
			name:    "direct miss",
			err:     base,
			targets: []error{New("other")},
			want:    false,
		},
		{
			name:    "no targets",
			err:     base,
			targets: nil,
			want:    false,
		},
		{
			name:    "wrap chain",
			err:     Wrap(base, "context"),
			targets: []error{base},
			want:    true,
		},
		{
			name:    "with message chain",
			err:     WithMessage(base, "message"),
			targets: []error{base},
			want:    true,
		},
		{
			name:    "with stack chain",
			err:     WithStack(base),
			targets: []error{base},
			want:    true,
		},
		{
			name:    "one of many targets",
			err:     Wrap(base, "context"),
			targets: []error{New("a"), base},
			want:    true,
		},
		{
			name:    "deep chain",
			err:     Wrap(WithStack(WithMessage(base, "message")), "context"),
			targets: []error{base},
			want:    true,
		},
		{
			name:    "stdlib error in chain",
			err:     Wrap(stdio.EOF, "context"),
			targets: []error{stdio.EOF},
			want:    true,
		},
		{
			name:    "stdlib error miss",
			err:     Wrap(stdio.EOF, "context"),
			targets: []error{stdio.ErrClosedPipe},
			want:    false,
		},
		{
			name:    "multi error match",
			err:     stderrors.Join(stdio.EOF, base),
			targets: []error{base},
			want:    true,
		},
		{
			name:    "multi error match first",
			err:     stderrors.Join(stdio.EOF, base),
			targets: []error{stdio.EOF},
			want:    true,
		},
		{
			name:    "multi error miss",
			err:     stderrors.Join(stdio.EOF, stdio.ErrClosedPipe),
			targets: []error{base},
			want:    false,
		},
		{
			name:    "multi error with nil",
			err:     stderrors.Join(nil, base),
			targets: []error{base},
			want:    true,
		},
		{
			name:    "custom is method",
			err:     &customIsError{target: base},
			targets: []error{base},
			want:    true,
		},
		{
			name:    "custom is method miss",
			err:     &customIsError{target: New("other")},
			targets: []error{base},
			want:    false,
		},
		{
			name:    "wrapped custom is method",
			err:     Wrap(&customIsError{target: base}, "context"),
			targets: []error{base},
			want:    true,
		},
		{
			name:    "unwrapping to target",
			err:     unwrappingError{err: base},
			targets: []error{base},
			want:    true,
		},
		{
			name:    "unwrapping miss",
			err:     unwrappingError{err: New("other")},
			targets: []error{base},
			want:    false,
		},
		{
			name:    "wrap of nil is nil",
			err:     Wrap(nil, "context"),
			targets: []error{base},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *stdtesting.T) {
			if got := Iss(tt.err, tt.targets...); got != tt.want {
				t.Errorf("Iss() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWrapMulti(t *stdtesting.T) {
	e1 := New("err1")
	e2 := NewCode(2, "err2")

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "no errors",
			err:  WrapMulti(nil, "message"),
		},
		{
			name: "all nil",
			err:  WrapMulti([]error{nil, nil}, "message"),
		},
		{
			name: "format no errors",
			err:  WrapMultif(nil, "message %s", "x"),
		},
		{
			name: "single",
			err:  WrapMulti([]error{e1}, "message"),
			want: "message: err1",
		},
		{
			name: "single with nil",
			err:  WrapMulti([]error{e1, nil}, "message"),
			want: "message: err1",
		},
		{
			name: "multiple",
			err:  WrapMulti([]error{e1, e2}, "message"),
			want: "message: err1\nerr2",
		},
		{
			name: "multiple with format",
			err:  WrapMultif([]error{e1, e2}, "message %d", 1),
			want: "message 1: err1\nerr2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *stdtesting.T) {
			if tt.want == "" {
				if tt.err != nil {
					t.Fatalf("expected nil error, got %v", tt.err)
				}

				return
			}
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestWrapMultiUnwrap проверяет, что мульти-ошибка строится на стандартном
// errors.Join и поддерживает Unwrap() []error: stdlib-функции errors.Is и
// errors.As проходят в под-ошибки.
func TestWrapMultiUnwrap(t *stdtesting.T) {
	e1 := New("err1")
	e2 := NewCode(2, "err2")

	t.Run("As reaches Unwrap() []error", func(t *stdtesting.T) {
		var multi interface{ Unwrap() []error }
		if !stderrors.As(WrapMulti([]error{e1, e2}, "m"), &multi) {
			t.Fatal("errors.As не нашла interface{ Unwrap() []error }")
		}
		subs := multi.Unwrap()
		if len(subs) != 2 {
			t.Fatalf("len(Unwrap()) = %d, want 2", len(subs))
		}
		// Порядок под-ошибок — как в stderrors.Join.
		if subs[0] != e1 || subs[1] != e2 {
			t.Errorf("порядок под-ошибок = %v, want [e1 e2]", subs)
		}
	})

	t.Run("Is finds sub-error", func(t *stdtesting.T) {
		if !stderrors.Is(WrapMulti([]error{nil, stdio.EOF}, "m"), stdio.EOF) {
			t.Error("errors.Is не нашёл io.EOF в мульти-ошибке")
		}
	})

	t.Run("As finds concrete type", func(t *stdtesting.T) {
		e := customError{msg: "x"}
		var target customError
		if !stderrors.As(WrapMulti([]error{e, stdio.EOF}, "m"), &target) {
			t.Fatal("errors.As не нашёл конкретный тип под-ошибки")
		}
		if target.msg != "x" {
			t.Errorf("target.msg = %q, want %q", target.msg, "x")
		}
	})
}

func TestWrapMultiCode(t *stdtesting.T) {
	e1 := New("err1")
	e2 := NewCode(2, "err2")

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "no errors",
			err:  WrapMultiCode(5, nil, "message"),
		},
		{
			name: "all nil",
			err:  WrapMultiCode(5, []error{nil}, "message"),
		},
		{
			name: "multiple",
			err:  WrapMultiCode(5, []error{e1, e2}, "message"),
			want: "message: err1\nerr2",
		},
		{
			name: "multiple with format",
			err:  WrapMultiCodef(5, []error{e1, e2}, "message %d", 1),
			want: "message 1: err1\nerr2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *stdtesting.T) {
			if tt.want == "" {
				if tt.err != nil {
					t.Fatalf("expected nil error, got %v", tt.err)
				}

				return
			}
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("codes", func(t *stdtesting.T) {
		err := WrapMultiCode(5, []error{e1, e2}, "message")
		if !IsCode(err, 5) {
			t.Error("IsCode did not find the wrap code")
		}
		if !IsCode(err, 2) {
			t.Error("IsCode did not find the inner code")
		}
		if IsCode(err, 99) {
			t.Error("IsCode found a code that is not in the chain")
		}
	})
}

func TestWrapMultiChain(t *stdtesting.T) {
	e1 := New("err1")
	e2 := NewCode(2, "err2")
	err := WrapMulti([]error{e1, e2}, "message")

	t.Run("Iss", func(t *stdtesting.T) {
		if !Iss(err, e1) {
			t.Error("Iss did not find the first error")
		}
		if !Iss(err, e2) {
			t.Error("Iss did not find the second error")
		}
		if Iss(err, New("other")) {
			t.Error("Iss found an error that is not in the chain")
		}
	})

	t.Run("IsCode", func(t *stdtesting.T) {
		if !IsCode(err, 2) {
			t.Error("IsCode did not find the code of the second error")
		}
		if IsCode(err, 99) {
			t.Error("IsCode found a code that is not in the chain")
		}
	})

	t.Run("stdlib Is", func(t *stdtesting.T) {
		if !stderrors.Is(WrapMulti([]error{stdio.EOF}, "message"), stdio.EOF) {
			t.Error("errors.Is did not find io.EOF")
		}
	})

	t.Run("Cause", func(t *stdtesting.T) {
		cause := Cause(err)
		if cause == nil {
			t.Fatal("Cause returned nil")
		}
		if got := cause.Error(); got != "err1\nerr2" {
			t.Errorf("Cause().Error() = %q, want %q", got, "err1\nerr2")
		}
	})

	t.Run("Unwrap", func(t *stdtesting.T) {
		// цепочка: withStack -> withMessage -> joinError; объединённая
		// ошибка — конец цепочки одиночных ошибок
		if got := Unwrap(Unwrap(err)); got == nil {
			t.Error("expected the joined error after two unwraps, got nil")
		}
		if got := Unwrap(Unwrap(Unwrap(err))); got != nil {
			t.Errorf("expected nil after three unwraps, got %v", got)
		}
	})

	t.Run("multi unwrap", func(t *stdtesting.T) {
		cause := Cause(err)
		multi, ok := cause.(interface{ Unwrap() []error })
		if !ok {
			t.Fatal("cause does not implement Unwrap() []error")
		}
		if got := len(multi.Unwrap()); got != 2 {
			t.Errorf("len(Unwrap()) = %d, want 2", got)
		}
	})

	t.Run("has stack", func(t *stdtesting.T) {
		if _, ok := err.(StackTracer); !ok {
			t.Error("WrapMulti result does not implement StackTracer")
		}
	})
}
