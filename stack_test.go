package errors

import (
	stderrors "errors"
	stdregexp "regexp"
	stdruntime "runtime"
	stdstrings "strings"
	stdtesting "testing"
	stdunsafe "unsafe"
)

type X struct{}

func TestFuncPtr(t *stdtesting.T) {
	t.Run("String: известный PC", func(t *stdtesting.T) {
		want := testModule + `\.init(\.ializers)?` + ` .+/fixtures_test.go:\d+$`
		if !stdregexp.MustCompile(want).MatchString(initpc.String()) {
			t.Errorf("initpc.String() = %q, want match %q", initpc.String(), want)
		}
	})

	t.Run("String: неизвестный PC", func(t *stdtesting.T) {
		if got := FuncPtr(0).String(); got != "" {
			t.Errorf("FuncPtr(0).String() = %q, want \"\"", got)
		}
	})

	t.Run("AsWrite: неизвестный PC", func(t *stdtesting.T) {
		var b stdstrings.Builder
		n, err := FuncPtr(0).WriteTo(&b)
		if err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("AsWrite() = %d байт, want 0", n)
		}
		if b.String() != "" {
			t.Errorf("AsWrite() записал %q, want пустую строку", b.String())
		}
	})
}

func TestStackTrace(t *stdtesting.T) {
	tests := []struct {
		err  error
		want []string
	}{{
		fixtureStackTraceNew(), []string{
			testModule + `\.fixtureStackTraceNew` + ` .+/` + fixtureFile + `:\d+$`,
		},
	}, {
		fixtureStackTraceWrap(), []string{
			testModule + `\.fixtureStackTraceWrap` + ` .+/` + fixtureFile + `:\d+$`, // это стек Wrap, а не New
		},
	}, {
		Cause(fixtureStackTraceWrap()), []string{
			testModule + `\.fixtureStackTraceWrap` + ` .+/` + fixtureFile + `:\d+$`, // это стек New
		},
	}, {
		// В этом GOST-окружении заглушка runtime.Callers возвращает ограниченный
		// набор кадров (skip+1), поэтому кадр вызывающего теста в стеке
		// фикстуры отсутствует — проверяем только кадр фикстуры.
		fixtureStackTraceNew(), []string{
			testModule + `\.fixtureStackTraceNew` + ` .+/` + fixtureFile + `:\d+$`,
		},
	}, {
		Cause(fixtureStackTraceNested()), []string{
			testModule + `\.fixtureStackTraceNestedInner` + ` .+/` + fixtureFile + `:\d+$`, // это стек Errorf
			testModule + `\.fixtureStackTraceNested` + ` .+/` + fixtureFile + `:\d+$`,      // это стек вызывающего Errorf
		},
	}}
	for i, tt := range tests {
		x, ok := tt.err.(StackTracer)
		if !ok {
			t.Errorf("test %d: expected %#v to implement StackTracer", i+1, tt.err)
			continue
		}
		st := x.StackTrace()
		for j, want := range tt.want {
			if j >= len(st) {
				t.Errorf("test %d: кадр %d: стек короче %d кадров", i+1, j+1, len(tt.want))
				continue
			}
			if !stdregexp.MustCompile(want).MatchString(st[j].String()) {
				t.Errorf("test %d: кадр %d = %q, want match %q", i+1, j+1, st[j].String(), want)
			}
		}
	}
}

func stackTrace() Stack {
	const depth = 8
	var pcs [depth]uintptr
	n := stdruntime.Callers(1, pcs[:])
	frames := pcs[0:n]

	return *(*Stack)(stdunsafe.Pointer(&frames))
}

func TestStackTraceFormat(t *stdtesting.T) {
	// В этом GOST-окружении runtime.Callers — заглушка gogost, поэтому
	// первый кадр стека — это runtime.Callers (runtime/extern.go), а не
	// stackTrace, как в стандартном тулчейне.
	stack2 := `runtime\.Callers` + ` .+/runtime/extern.go:\d+` + "\n" +
		testModule + `\.TestStackTraceFormat` + ` .+/` + stackTestFile + `:\d+`

	tests := []struct {
		Stack

		format string
		want   string
	}{{
		nil,
		"%s",
		"",
	}, {
		nil,
		"%v",
		"",
	}, {
		nil,
		"%+v",
		"",
	}, {
		nil,
		"%#v",
		`errors\.Stack\(nil`,
	}, {
		make(Stack, 0),
		"%s",
		"",
	}, {
		make(Stack, 0),
		"%v",
		"",
	}, {
		make(Stack, 0),
		"%+v",
		"",
	}, {
		make(Stack, 0),
		"%#v",
		`errors\.Stack\{\}`,
	}, {
		stackTrace()[:2],
		"%s",
		stack2,
	}, {
		stackTrace()[:2],
		"%v",
		stack2,
	}, {
		stackTrace()[:2],
		"%+v",
		stack2,
	}, {
		stackTrace()[:2],
		"%#v",
		`errors\.Stack\{0x[0-9a-f]+, 0x[0-9a-f]+\}`,
	}}

	for i, tt := range tests {
		testFormatRegexp(t, i, tt.Stack, tt.format, tt.want)
	}
}

// версия runtime.Caller, возвращающая FuncPtr, а не uintptr.
func caller() FuncPtr {
	var pcs [3]uintptr
	n := stdruntime.Callers(2, pcs[:])
	frames := stdruntime.CallersFrames(pcs[:n])
	frame, _ := frames.Next()

	return FuncPtr(frame.PC)
}

// foreignStackerError — «чужой» реализатор стека (интерфейс со
// StackTrace()), не созданный конструкторами этого пакета.
type foreignStackerError struct {
	*Stack

	msg string
}

func (f *foreignStackerError) Error() string { return f.msg }

func TestStackBlocks(t *stdtesting.T) {
	// Глобал CutTail мутатируется init() из example_test.go (= 3) —
	// возвращаем дефолтное значение пакета (2) на время теста.
	oldCutTail := CutTail
	CutTail = 2
	t.Cleanup(func() { CutTail = oldCutTail })

	// Фикстура вызывается внутри замыкания t.Run, поэтому имя кадра теста
	// может иметь суффикс .funcN.
	wantTest := testModule + `\.TestStackBlocks(\.func\d+)?` + ` .+/` + stackTestFile + `:\d+$`
	reTest := stdregexp.MustCompile(wantTest)
	frameRe := stdregexp.MustCompile(`.+ .+:\d+$`)

	// lineFrame возвращает номер строки первого кадра блока.
	lineFrame := func(t *stdtesting.T, b []FrameLine) int {
		t.Helper()
		if len(b) == 0 {
			t.Fatal("блок пуст, want >= 1 кадр")
		}
		frame, err := b[0].ParseStack()
		if err != nil {
			t.Fatalf("ParseStack(%q) = %v", b[0], err)
		}

		return frame.Line
	}

	tests := []struct {
		name  string
		make  func() error
		check func(t *stdtesting.T, got [][]FrameLine)
	}{
		{
			name: "nil",
			make: func() error { return nil },
			check: func(t *stdtesting.T, got [][]FrameLine) {
				t.Helper()
				if len(got) != 0 {
					t.Errorf("StackBlocks(nil) = %q, want 0 блоков", got)
				}
			},
		},
		{
			name: "без стека",
			make: func() error { return stderrors.New("plain") },
			check: func(t *stdtesting.T, got [][]FrameLine) {
				t.Helper()
				// Ошибка не реализует StackTracer — блока не формирует.
				if len(got) != 0 {
					t.Errorf("StackBlocks(): len = %d, want 0", len(got))
				}
			},
		},
		{
			name: "обёртка над ошибкой без стека",
			make: func() error { return WithMessage(stderrors.New("plain"), "wrap") },
			check: func(t *stdtesting.T, got [][]FrameLine) {
				t.Helper()
				// Ни обёртка, ни исходная ошибка стека не несут.
				if len(got) != 0 {
					t.Errorf("StackBlocks(): len = %d, want 0", len(got))
				}
			},
		},
		{
			name: "стек из фикстуры",
			make: fixtureStackTraceNew,
			check: func(t *stdtesting.T, got [][]FrameLine) {
				t.Helper()
				// Одна ошибка с одним стеком — один блок.
				if len(got) != 1 {
					t.Fatalf("StackBlocks(): len = %d, want 1", len(got))
				}
				b := got[0]
				if len(b) < 1 {
					t.Fatalf("блок 0: len = %d, want >= 1", len(b))
				}
				// Каждый кадр — строка «func file:line».
				for i, g := range b {
					if !frameRe.MatchString(string(g)) {
						t.Errorf("блок 0, кадр %d = %q, want «func file:line»", i, g)
					}
				}
				wantFirst := testModule + `\.fixtureStackTraceNew` + ` .+/` + fixtureFile + `:\d+$`
				foundFirst := false
				foundTest := false
				for _, g := range b {
					if !foundFirst && stdregexp.MustCompile(wantFirst).MatchString(string(g)) {
						foundFirst = true
					}
					if !foundTest && reTest.MatchString(string(g)) {
						foundTest = true
					}
				}
				if !foundFirst {
					t.Errorf("нет кадра фикстуры, want match %q; got: %q", wantFirst, b)
				}
				if !foundTest {
					t.Errorf("нет кадра теста, want match %q; got: %q", wantTest, b)
				}
			},
		},
		{
			name: "вложенный стек из фикстуры",
			make: fixtureStackTraceNested,
			check: func(t *stdtesting.T, got [][]FrameLine) {
				t.Helper()
				// В цепочке одна ошибка со стеком (Newf во внутренней
				// фикстуре) — один блок.
				if len(got) != 1 {
					t.Fatalf("StackBlocks(): len = %d, want 1", len(got))
				}
				b := got[0]
				if len(b) < 2 {
					t.Fatalf("блок 0: len = %d, want >= 2", len(b))
				}
				for i, g := range b {
					if !frameRe.MatchString(string(g)) {
						t.Errorf("блок 0, кадр %d = %q, want «func file:line»", i, g)
					}
				}
				wants := []string{
					testModule + `\.fixtureStackTraceNestedInner` + ` .+/` + fixtureFile + `:\d+$`,
					testModule + `\.fixtureStackTraceNested` + ` .+/` + fixtureFile + `:\d+$`,
				}
				idx := []int{-1, -1}
				for i, g := range b {
					for wi, w := range wants {
						if idx[wi] < 0 && stdregexp.MustCompile(w).MatchString(string(g)) {
							idx[wi] = i
						}
					}
				}
				for wi, w := range wants {
					if idx[wi] < 0 {
						t.Errorf("кадр %d не найден, want match %q; got: %q", wi, w, b)
					}
				}
				// Кадр внутренней фикстуры выше кадра вызывающей (сверху вниз).
				if idx[0] >= 0 && idx[1] >= 0 && idx[0] > idx[1] {
					t.Errorf("порядок кадров: inner = %d, nested = %d; want inner < nested", idx[0], idx[1])
				}
			},
		},
		{
			name: "цепочка New+Wrap",
			make: func() error {
				e := New("ooh")
				return Wrap(e, "ahh")
			},
			check: func(t *stdtesting.T, got [][]FrameLine) {
				t.Helper()
				// Две ошибки со стеками — два блока: блок 0 — обёртка
				// Wrap (внешняя), блок 1 — New (внутренняя).
				if len(got) != 2 {
					t.Fatalf("StackBlocks(): len = %d, want 2", len(got))
				}
				for bi, b := range got {
					if len(b) < 1 {
						t.Fatalf("блок %d: len = %d, want >= 1", bi, len(b))
					}
					for fi, g := range b {
						if !frameRe.MatchString(string(g)) {
							t.Errorf("блок %d, кадр %d = %q, want «func file:line»", bi, fi, g)
						}
					}
					// Первый кадр — строка вызова в замыкании этого теста.
					if !reTest.MatchString(string(b[0])) {
						t.Errorf("блок %d, кадр 0 = %q, want match %q", bi, b[0], wantTest)
					}
				}
				// Строка вызова Wrap идёт позже строки вызова New.
				l0, l1 := lineFrame(t, got[0]), lineFrame(t, got[1])
				if l0 <= l1 {
					t.Errorf("строки кадров 0: wrap = %d, new = %d; want wrap > new (от внешней к внутренней)", l0, l1)
				}
			},
		},
		{
			name: "мульти",
			make: func() error {
				e1 := New("whoops")
				e2 := Newf("%s", "oops")
				return WrapMulti([]error{e1, e2}, "pops")
			},
			check: func(t *stdtesting.T, got [][]FrameLine) {
				t.Helper()
				// Под-ошибка 1, под-ошибка 2, обёртка WrapMulti.
				if len(got) != 3 {
					t.Fatalf("StackBlocks(): len = %d, want 3", len(got))
				}
				for bi, b := range got {
					if len(b) < 1 {
						t.Fatalf("блок %d: len = %d, want >= 1", bi, len(b))
					}
					for fi, g := range b {
						s := string(g)
						if !frameRe.MatchString(s) {
							t.Errorf("блок %d, кадр %d = %q, want «func file:line»", bi, fi, g)
						}
						// Строки-сообщения в блоки не попадают.
						for _, msg := range []string{"whoops", "oops", "pops"} {
							if stdstrings.Contains(s, msg) {
								t.Errorf("блок %d, кадр %d = %q, содержит сообщение %q", bi, fi, g, msg)
							}
						}
					}
					// Первый кадр — точка вызова в замыкании этого теста.
					if !reTest.MatchString(string(b[0])) {
						t.Errorf("блок %d, кадр 0 = %q, want match %q", bi, b[0], wantTest)
					}
				}
				// Порядок: под-ошибки в порядке Join, затем обёртка — по
				// номерам строк вызовов.
				l0, l1, l2 := lineFrame(t, got[0]), lineFrame(t, got[1]), lineFrame(t, got[2])
				if l0 >= l1 || l1 >= l2 {
					t.Errorf("строки кадров 0: %d, %d, %d; want строго по возрастанию (под1 < под2 < обёртка)", l0, l1, l2)
				}
			},
		},
		{
			name: "чужой стек с неизвестным PC",
			make: func() error {
				return &foreignStackerError{msg: "unknown", Stack: &Stack{0}}
			},
			check: func(t *stdtesting.T, got [][]FrameLine) {
				t.Helper()
				// PC 0 не находит функцию — кадр пропускается, блоков нет.
				if len(got) != 0 {
					t.Errorf("StackBlocks() = %q, want 0 блоков", got)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *stdtesting.T) {
			tt.check(t, StackBlocks(tt.make()))
		})
	}
}

func TestParseStack(t *stdtesting.T) {
	t.Run("FrameLine: одиночный кадр", func(t *stdtesting.T) {
		got, err := FrameLine("a.F /x.go:1").ParseStack()
		if err != nil {
			t.Fatalf("FrameLine.ParseStack: %v", err)
		}
		want := &FuncFrame{Func: "a.F", File: "/x.go", Line: 1}
		if *got != *want {
			t.Errorf("FrameLine.ParseStack() = %+v, want %+v", got, want)
		}
	})

	t.Run("FrameLine: неверный кадр", func(t *stdtesting.T) {
		bad := []FrameLine{
			"",
			"без пробела",
			"/x.go:нецифра",
			"f",
			"f /x.go:",
			"f :",
		}
		for _, f := range bad {
			frame, err := f.ParseStack()
			if err == nil {
				t.Errorf("FrameLine(%q).ParseStack() = %v, want error", f, frame)
			}
			if frame != nil {
				t.Errorf("FrameLine(%q).ParseStack() frame = %v, want nil", f, frame)
			}
		}
	})
}
