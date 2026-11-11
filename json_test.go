package errors

import (
	"encoding/json"
	stderrors "errors"
	stdregexp "regexp"
	stdstrings "strings"
	stdtesting "testing"
)

func TestFuncFrameMarshalJSON(t *stdtesting.T) {
	got, err := json.Marshal(FuncFrame{File: "x.go", Line: 42, Func: "f"})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"source":"x.go","line":42,"func":"f"}` {
		t.Errorf("json.Marshal(FuncFrame) = %s", got)
	}
}

// TestMarshalJSONErrors проверяет, что json.Marshal ошибки пакета
// сериализует её как JSON-строку с текстом Error(), а не внутреннее
// тело структуры.
func TestMarshalJSONErrors(t *stdtesting.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "New",
			err:  New("hello"),
			want: `"hello"`,
		},
		{
			name: "WithMessage",
			err:  WithMessage(New("root"), "ctx"),
			want: `"ctx: root"`,
		},
		{
			name: "Wrap",
			err:  Wrap(New("root"), "ctx"),
			want: `"ctx: root"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *stdtesting.T) {
			got, err := json.Marshal(tt.err)
			if err != nil {
				t.Fatalf("json.Marshal: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("json.Marshal = %s, want %s", got, tt.want)
			}
		})
	}

	t.Run("структура-носитель", func(t *stdtesting.T) {
		var r struct {
			Err error `json:"err"`
		}
		r.Err = New("x")
		got, err := json.Marshal(r)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		if string(got) != `{"err":"x"}` {
			t.Errorf("json.Marshal = %s, want %s", got, `{"err":"x"}`)
		}
	})
}

func TestMarshalStackError(t *stdtesting.T) {
	tests := []struct {
		name  string
		err   error
		check func(t *stdtesting.T, err error, got any)
	}{
		{
			name: "nil",
			err:  nil,
			check: func(t *stdtesting.T, _ error, got any) {
				t.Helper()
				if got != nil {
					t.Errorf("MarshalStackError(nil) = %v, want nil", got)
				}
			},
		},
		{
			name: "без стека",
			err:  stderrors.New("plain"),
			check: func(t *stdtesting.T, _ error, got any) {
				t.Helper()
				blocks, ok := got.([][]FrameLine)
				if !ok {
					t.Fatalf("MarshalStackError() = %T, want [][]FrameLine", got)
				}
				if len(blocks) != 0 {
					t.Errorf("len = %d, want 0", len(blocks))
				}
			},
		},
		{
			name: "со стеком",
			err:  fixtureStackTraceNew(),
			check: func(t *stdtesting.T, err error, got any) {
				t.Helper()
				blocks, ok := got.([][]FrameLine)
				if !ok {
					t.Fatalf("MarshalStackError() = %T, want [][]FrameLine", got)
				}
				// Одна ошибка со стеком — один блок.
				if len(blocks) != 1 {
					t.Fatalf("len = %d, want 1", len(blocks))
				}
				stack := blocks[0]
				// По одному кадру на кадр стека той же ошибки.
				wantLen := len(err.(StackTracer).StackTrace())
				if len(stack) != wantLen {
					t.Fatalf("len = %d, want %d", len(stack), wantLen)
				}
				// Каждый кадр — «func file:line» без \n и \t.
				for i, s := range stack {
					if !stdregexp.MustCompile(`.+ .+:\d+$`).MatchString(string(s)) {
						t.Errorf("стек[0][%d] = %q, want кадр «func file:line»", i, s)
					}
					if stdstrings.Contains(string(s), "\n") || stdstrings.Contains(string(s), "\t") {
						t.Errorf("стек[0][%d] = %q, не должен содержать \\n или \\t", i, s)
					}
				}
				want := testModule + `\.fixtureStackTraceNew` + ` .+/` + fixtureFile + `:\d+$`
				if !stdregexp.MustCompile(want).MatchString(string(stack[0])) {
					t.Errorf("стек[0][0] = %q, want match %q", stack[0], want)
				}
				// Результат должен быть JSON-сериализуемым: вложенный
				// массив массивов строк [["..."],["..."]].
				b, err := json.Marshal(got)
				if err != nil {
					t.Fatalf("json.Marshal: %v", err)
				}
				var arr [][]FrameLine
				if err := json.Unmarshal(b, &arr); err != nil {
					t.Fatalf("json.Unmarshal: %v", err)
				}
				if len(arr) != len(blocks) {
					t.Fatalf("len(JSON) = %d, want %d", len(arr), len(blocks))
				}
				for bi := range arr {
					if len(arr[bi]) != len(blocks[bi]) {
						t.Errorf("len(JSON[%d]) = %d, want %d", bi, len(arr[bi]), len(blocks[bi]))
					}
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *stdtesting.T) {
			tt.check(t, tt.err, MarshalStackError(tt.err))
		})
	}

	t.Run("мульти", func(t *stdtesting.T) {
		e1 := New("whoops")
		e2 := Newf("%s", "oops")
		m := WrapMulti([]error{e1, e2}, "pops")
		got := MarshalStackError(m)
		blocks, ok := got.([][]FrameLine)
		if !ok {
			t.Fatalf("MarshalStackError() = %T, want [][]FrameLine", got)
		}
		// Блоки: первая под-ошибка, вторая под-ошибка, обёртка WrapMulti.
		if len(blocks) != 3 {
			t.Fatalf("len = %d, want 3\ngot: %q", len(blocks), blocks)
		}
		for bi, b := range blocks {
			if len(b) < 1 {
				t.Fatalf("блок %d: len = %d, want >= 1", bi, len(b))
			}
			for i, s := range b {
				if !stdregexp.MustCompile(`.+ .+:\d+$`).MatchString(string(s)) {
					t.Errorf("стек[%d][%d] = %q, want кадр «func file:line»", bi, i, s)
				}
				// Строки-сообщения в блоки не попадают.
				for _, msg := range []string{"whoops", "oops", "pops"} {
					if stdstrings.Contains(string(s), msg) {
						t.Errorf("стек[%d][%d] = %q, содержит сообщение %q", bi, i, s, msg)
					}
				}
				if stdstrings.Contains(string(s), "\n") || stdstrings.Contains(string(s), "\t") {
					t.Errorf("стек[%d][%d] = %q, не должен содержать \\n или \\t", bi, i, s)
				}
			}
		}
	})
}
