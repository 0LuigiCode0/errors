package errors_test

import (
	stdbytes "bytes"
	"encoding/json"
	stdio "io"
	stdos "os"
	stdregexp "regexp"
	stdstrconv "strconv"
	stdstrings "strings"
	stdtesting "testing"

	"github.com/0LuigiCode0/errors"
	"github.com/rs/zerolog"
)

// TestZerologErrorStackMarshaler проверяет, что MarshalStackError
// подходит на роль zerolog.ErrorStackMarshaler: поле "stack" в JSON
// лога — вложенный массив блоков кадров, каждый внутренний массив —
// кадры «func file:line» одной ошибки цепочки, без «\n»/«\t» внутри
// (в raw-JSON экспайзы отсутствуют).
//
// В zerolog v1.35.1 Event.Err() читает флажок e.stack, поэтому Stack()
// вызывается ДО Err().
func TestZerologErrorStackMarshaler(t *stdtesting.T) {
	old := zerolog.ErrorStackMarshaler
	zerolog.ErrorStackMarshaler = errors.MarshalStackError
	t.Cleanup(func() { zerolog.ErrorStackMarshaler = old })

	var buf stdbytes.Buffer
	logger := zerolog.New(stdio.MultiWriter(&buf, stdos.Stdout))

	cause := errors.New("whoops")
	err := errors.Wrap(cause, "read failed")
	logger.Error().Stack().Err(err).Msg("boom")

	var entry struct {
		Level string          `json:"level"`
		Error string          `json:"error"`
		Msg   string          `json:"message"`
		Stack json.RawMessage `json:"stack"`
	}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, buf.String())
	}
	if entry.Level != "error" {
		t.Errorf("level = %q, want %q", entry.Level, "error")
	}
	if entry.Error != "read failed: whoops" {
		t.Errorf("error = %q, want %q", entry.Error, "read failed: whoops")
	}
	if len(entry.Stack) == 0 || string(entry.Stack) == "null" {
		t.Fatalf("stack = %s, want непустой", entry.Stack)
	}

	// Поле "stack" — вложенный JSON-массив блоков кадров: блок 0 —
	// стек обёртки Wrap, блок 1 — стек исходной ошибки New.
	var stackArr [][]string
	if err := json.Unmarshal(entry.Stack, &stackArr); err != nil {
		t.Fatalf("unmarshal stack: %v\nraw: %s", err, entry.Stack)
	}
	if len(stackArr) != 2 {
		t.Fatalf("len(stack) = %d, want 2; raw: %s", len(stackArr), entry.Stack)
	}
	// Каждый элемент каждого блока — кадр «func file:line», без «\n»/«\t».
	frameRe := stdregexp.MustCompile(`.+ .+:\d+$`)
	for bi, block := range stackArr {
		if len(block) < 1 {
			t.Fatalf("stack[%d] пуст; raw: %s", bi, entry.Stack)
		}
		for i, s := range block {
			if !frameRe.MatchString(s) {
				t.Errorf("stack[%d][%d] = %q, want кадр «func file:line»", bi, i, s)
			}
			if stdstrings.Contains(s, "\n") || stdstrings.Contains(s, "\t") {
				t.Errorf("stack[%d][%d] = %q, не должен содержать \\n или \\t", bi, i, s)
			}
		}
	}
	// Первый кадр первого блока — точка вызова в этом тесте.
	if !stdstrings.Contains(stackArr[0][0], "TestZerologErrorStackMarshaler") {
		t.Errorf("stack[0][0] = %q, не содержит имя функции теста", stackArr[0][0])
	}
	if !stdstrings.Contains(stackArr[0][0], "zerolog_test.go:") {
		t.Errorf("stack[0][0] = %q, не содержит zerolog_test.go:", stackArr[0][0])
	}

	raw := string(entry.Stack)
	// Экспайзы «\n»/«\t» в raw-JSON отсутствуют.
	if stdstrings.Contains(raw, `\n`) {
		t.Errorf("raw-JSON содержит экспайз «\\n»: %s", raw)
	}
	if stdstrings.Contains(raw, `\t`) {
		t.Errorf("raw-JSON содержит экспайз «\\t»: %s", raw)
	}
	// Раньше поле содержало JSON-объекты с ключами err/frames — их больше нет.
	if stdstrings.Contains(raw, `"err"`) {
		t.Errorf("stack содержит устаревший ключ \"err\": %s", raw)
	}
	if stdstrings.Contains(raw, `"frames"`) {
		t.Errorf("stack содержит устаревший ключ \"frames\": %s", raw)
	}
}

// TestZerologErrorStackMarshalerMulti проверяет, что для мульти-ошибки
// поле "stack" — вложенный массив блоков кадров: сначала блоки под-ошибок
// в порядке Join, затем блок обёртки WrapMulti; строки-сообщения в
// "stack" не попадают (сообщение остаётся в поле "error"); экспайзы
// «\n»/«\t» в raw-JSON отсутствуют.
func TestZerologErrorStackMarshalerMulti(t *stdtesting.T) {
	old := zerolog.ErrorStackMarshaler
	zerolog.ErrorStackMarshaler = errors.MarshalStackError
	t.Cleanup(func() { zerolog.ErrorStackMarshaler = old })

	var buf stdbytes.Buffer
	logger := zerolog.New(stdio.MultiWriter(&buf, stdos.Stdout))

	err1 := errors.New("whoops")
	err2 := errors.Newf("%s", "oops")
	m := errors.WrapMulti([]error{err1, err2}, "pops")
	logger.Error().Stack().Err(m).Msg("boom")

	var entry struct {
		Level string          `json:"level"`
		Error string          `json:"error"`
		Msg   string          `json:"message"`
		Stack json.RawMessage `json:"stack"`
	}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, buf.String())
	}
	if entry.Level != "error" {
		t.Errorf("level = %q, want %q", entry.Level, "error")
	}
	// Поле "error" несёт сообщение мульти-ошибки — оно не дублируется
	// в "stack".
	if entry.Error != "pops: whoops\noops" {
		t.Errorf("error = %q, want %q", entry.Error, "pops: whoops\noops")
	}
	if len(entry.Stack) == 0 || string(entry.Stack) == "null" {
		t.Fatalf("stack = %s, want непустой", entry.Stack)
	}

	var stackArr [][]string
	if err := json.Unmarshal(entry.Stack, &stackArr); err != nil {
		t.Fatalf("unmarshal stack: %v\nraw: %s", err, entry.Stack)
	}
	// Блоки: первая под-ошибка, вторая под-ошибка, обёртка WrapMulti.
	if len(stackArr) != 3 {
		t.Fatalf("len(stack) = %d, want 3; raw: %s", len(stackArr), entry.Stack)
	}
	frameRe := stdregexp.MustCompile(`.+ .+:\d+$`)
	for bi, block := range stackArr {
		if len(block) < 1 {
			t.Fatalf("stack[%d] пуст; raw: %s", bi, entry.Stack)
		}
		for i, s := range block {
			if !frameRe.MatchString(s) {
				t.Errorf("stack[%d][%d] = %q, want кадр «func file:line»", bi, i, s)
			}
			// Строки-сообщения в "stack" не попадают.
			for _, msg := range []string{"pops: whoops", "oops", "1. whoops", "2. oops"} {
				if stdstrings.Contains(s, msg) {
					t.Errorf("stack[%d][%d] = %q, содержит сообщение %q", bi, i, s, msg)
				}
			}
			if stdstrings.Contains(s, "\n") || stdstrings.Contains(s, "\t") {
				t.Errorf("stack[%d][%d] = %q, не должен содержать \\n или \\t", bi, i, s)
			}
		}
	}
	// Первый кадр каждого блока — строка вызова в этом тесте; строки
	// идут в порядке источников: под1 < под2 < обёртка.
	lines := make([]int, len(stackArr))
	for bi, block := range stackArr {
		s := block[0]
		if !stdstrings.Contains(s, "TestZerologErrorStackMarshalerMulti") {
			t.Errorf("stack[%d][0] = %q, не содержит имя функции теста", bi, s)
		}
		if !stdstrings.Contains(s, "zerolog_test.go:") {
			t.Errorf("stack[%d][0] = %q, не содержит zerolog_test.go:", bi, s)
		}
		tail := s[stdstrings.LastIndex(s, ":")+1:]
		ln, err := stdstrconv.Atoi(tail)
		if err != nil {
			t.Fatalf("stack[%d][0] = %q, не удалось разобрать номер строки: %v", bi, s, err)
		}
		lines[bi] = ln
	}
	if lines[0] >= lines[1] || lines[1] >= lines[2] {
		t.Errorf("строки кадров 0: %d, %d, %d; want по возрастанию (под1 < под2 < обёртка)", lines[0], lines[1], lines[2])
	}
	// Экспайзы «\n»/«\t» в raw-JSON отсутствуют.
	raw := string(entry.Stack)
	if stdstrings.Contains(raw, `\n`) {
		t.Errorf("raw-JSON содержит экспайз «\\n»: %s", raw)
	}
	if stdstrings.Contains(raw, `\t`) {
		t.Errorf("raw-JSON содержит экспайз «\\t»: %s", raw)
	}
}
