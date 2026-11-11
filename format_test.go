package errors

import (
	stdfmt "fmt"
	stdregexp "regexp"
	stdstrings "strings"
	stdtesting "testing"
)

func TestFormatNew(t *stdtesting.T) {
	tests := []struct {
		error

		format string
		want   string
	}{{
		fixtureNew(),
		"%s",
		"error",
	}, {
		fixtureNew(),
		"%v",
		"error",
	}, {
		fixtureNew(),
		"%+v",
		"error",
	}, {
		fixtureNew(),
		"%q",
		`"error"`,
	}}

	for i, tt := range tests {
		testFormatRegexp(t, i, tt.error, tt.format, tt.want)
	}
}

func TestFormatErrorf(t *stdtesting.T) {
	tests := []struct {
		error

		format string
		want   string
	}{{
		fixtureNewf(),
		"%s",
		"error",
	}, {
		fixtureNewf(),
		"%v",
		"error",
	}, {
		fixtureNewf(),
		"%+v",
		"error",
	}}

	for i, tt := range tests {
		testFormatRegexp(t, i, tt.error, tt.format, tt.want)
	}
}

func TestFormatWrap(t *stdtesting.T) {
	tests := []struct {
		error

		format string
		want   string
	}{{
		fixtureWrapNew(),
		"%s",
		"error2: error",
	}, {
		fixtureWrapNew(),
		"%v",
		"error2: error",
	}, {
		fixtureWrapNew(),
		"%+v",
		"error2: error",
	}, {
		fixtureWrapEOF(),
		"%s",
		"error: EOF",
	}, {
		fixtureWrapEOF(),
		"%v",
		"error: EOF",
	}, {
		fixtureWrapEOF(),
		"%+v",
		"error: EOF",
	}, {
		fixtureDoubleWrap(),
		"%+v",
		"error2: error1: EOF",
	}, {
		fixtureWrapContext(),
		"%q",
		`"context: error with space"`,
	}}

	for i, tt := range tests {
		testFormatRegexp(t, i, tt.error, tt.format, tt.want)
	}
}

func TestFormatWrapf(t *stdtesting.T) {
	tests := []struct {
		error

		format string
		want   string
	}{{
		fixtureWrapfEOF(),
		"%s",
		"error2: EOF",
	}, {
		fixtureWrapfEOF(),
		"%v",
		"error2: EOF",
	}, {
		fixtureWrapfEOF(),
		"%+v",
		"error2: EOF",
	}, {
		fixtureWrapfNew(),
		"%s",
		"error2: error",
	}, {
		fixtureWrapfNew(),
		"%v",
		"error2: error",
	}, {
		fixtureWrapfNew(),
		"%+v",
		"error2: error",
	}}

	for i, tt := range tests {
		testFormatRegexp(t, i, tt.error, tt.format, tt.want)
	}
}

func TestFormatWithStack(t *stdtesting.T) {
	tests := []struct {
		error

		format string
		want   string
	}{{
		fixtureWithStackEOF(),
		"%s",
		"EOF",
	}, {
		fixtureWithStackEOF(),
		"%v",
		"EOF",
	}, {
		fixtureWithStackEOF(),
		"%+v",
		"EOF",
	}, {
		fixtureWithStackNew(),
		"%s",
		"error",
	}, {
		fixtureWithStackNew(),
		"%v",
		"error",
	}, {
		fixtureWithStackNew(),
		"%+v",
		"error",
	}, {
		fixtureDoubleWithStack(),
		"%+v",
		"EOF",
	}, {
		fixtureWithStackWrapfEOF(),
		"%+v",
		"message: EOF",
	}, {
		fixtureWithStackNewf(),
		"%+v",
		"error1",
	}}

	for i, tt := range tests {
		testFormatRegexp(t, i, tt.error, tt.format, tt.want)
	}
}

func TestFormatWithMessage(t *stdtesting.T) {
	tests := []struct {
		error

		format string
		want   string
	}{{
		fixtureWithMessageNew(),
		"%s",
		"error2: error",
	}, {
		fixtureWithMessageNew(),
		"%v",
		"error2: error",
	}, {
		fixtureWithMessageNew(),
		"%+v",
		"error2: error",
	}, {
		fixtureWithMessageEOF(),
		"%s",
		"addition1: EOF",
	}, {
		fixtureWithMessageEOF(),
		"%v",
		"addition1: EOF",
	}, {
		fixtureWithMessageEOF(),
		"%+v",
		"addition1: EOF",
	}, {
		fixtureDoubleWithMessage(),
		"%v",
		"addition2: addition1: EOF",
	}, {
		fixtureDoubleWithMessage(),
		"%+v",
		"addition2: addition1: EOF",
	}, {
		fixtureWrapOfWithMessage(),
		"%+v",
		"error2: error1: EOF",
	}, {
		fixtureWithMessageNewf(),
		"%+v",
		"error2: error1",
	}, {
		fixtureWithMessageWithStack(),
		"%+v",
		"error: EOF",
	}, {
		fixtureWithMessageWrapWithStack(),
		"%+v",
		"outside-error: inside-error: EOF",
	}}

	for i, tt := range tests {
		testFormatRegexp(t, i, tt.error, tt.format, tt.want)
	}
}

func TestFormatGeneric(t *stdtesting.T) {
	starts := []struct {
		err  error
		want string
	}{
		{
			fixtureGenericNew(),
			"new-error",
		}, {
			fixtureGenericNewf(),
			"errorf-error",
		}, {
			fixtureGenericStdlibNew(),
			"errors-new-error",
		},
	}

	// want — вклад обёртки в сообщение: WithStack сообщения не меняет.
	wrappers := []wrapper{
		{
			fixtureWrapWithMessage,
			"with-message",
		}, {
			fixtureWrapWithStack,
			"",
		}, {
			fixtureWrapWrap,
			"wrap-error",
		}, {
			fixtureWrapWrapf,
			"wrapf-error1",
		},
	}

	for s := range starts {
		err := starts[s].err
		want := starts[s].want
		testFormatRegexp(t, s, err, "%+v", want)
		testGenericRecursive(t, err, want, wrappers, 3)
	}
}

func TestFormatWrappedNew(t *stdtesting.T) {
	tests := []struct {
		error

		format string
		want   string
	}{{
		fixtureWrappedNew("error"),
		"%+v",
		"error",
	}}

	for i, tt := range tests {
		testFormatRegexp(t, i, tt.error, tt.format, tt.want)
	}
}

func testFormatRegexp(t *stdtesting.T, n int, arg any, format, want string) {
	t.Helper()
	got := stdfmt.Sprintf(format, arg)
	gotLines := stdstrings.Split(got, "\n")
	wantLines := stdstrings.Split(want, "\n")

	if len(wantLines) > len(gotLines) {
		t.Errorf("test %d: wantLines(%d) > gotLines(%d):\n got: %q\nwant: %q", n+1, len(wantLines), len(gotLines), got, want)
		return
	}

	for i, w := range wantLines {
		match, err := stdregexp.MatchString(w, gotLines[i])
		if err != nil {
			t.Fatal(err)
		}
		if !match {
			t.Errorf("test %d: line %d: fmt.Sprintf(%q, err):\n got: %q\nwant: %q", n+1, i+1, format, got, want)
		}
	}
}

type wrapper struct {
	wrap func(err error) error
	want string
}

func testGenericRecursive(t *stdtesting.T, beforeErr error, beforeWant string, list []wrapper, maxDepth int) {
	t.Helper()
	if beforeWant == "" {
		panic("beforeWant must not be empty")
	}
	for _, w := range list {
		err := w.wrap(beforeErr)

		// WithStack сообщения не меняет, обёртка с сообщением добавляет
		// префикс «msg: ».
		want := beforeWant
		if w.want != "" {
			want = w.want + ": " + want
		}

		testFormatRegexp(t, maxDepth, err, "%+v", want)
		if maxDepth > 0 {
			testGenericRecursive(t, err, want, list, maxDepth-1)
		}
	}
}
