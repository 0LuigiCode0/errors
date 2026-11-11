package errors

import (
	stderrors "errors"
	stdfmt "fmt"
	stdio "io"
	stdruntime "runtime"
	stdstrconv "strconv"
	stdstrings "strings"
	stdunsafe "unsafe"
)

// FuncPtr представляет указатель на программу внутри кадра стека.
type FuncPtr uintptr

func (f FuncPtr) String() string {
	fun := stdruntime.FuncForPC(uintptr(f))
	if fun == nil {
		return ""
	}
	file, line := fun.FileLine(uintptr(f))

	return stdfmt.Sprintf("%s %s:%d", fun.Name(), file, line)
}

func (f FuncPtr) WriteTo(w stdio.Writer) (int64, error) {
	fun := stdruntime.FuncForPC(uintptr(f))
	if fun == nil {
		return 0, nil
	}
	file, line := fun.FileLine(uintptr(f))
	n, err := stdfmt.Fprintf(w, "%s %s:%d", fun.Name(), file, line)

	return int64(n), err
}

// StackTracer — ошибка, несущая трассировку стека.
type StackTracer interface {
	StackTrace() Stack
}

// Stack — стек точек вызова (указателей на программу). Реализует StackTracer и Stringer.
type Stack []FuncPtr

func (s Stack) StackTrace() Stack { return s }

func (s Stack) String() string {
	b := &stdstrings.Builder{}
	for _, ff := range s {
		if _, err := ff.WriteTo(b); err != nil {
			return b.String()
		}
		b.WriteByte('\n')
	}

	return b.String()
}

func (s Stack) WriteTo(w stdio.Writer) (int64, error) {
	var n int64
	for _, ff := range s {
		subN, err := ff.WriteTo(w)
		if err != nil {
			return n, err
		}
		n += subN
		if _, err := w.Write([]byte{'\n'}); err != nil {
			return n, err
		}
		n++
	}

	return n, nil
}

// FuncFrame — одна точка вызова в трассировке стека.
type FuncFrame struct {
	File string `json:"source"`
	Line int    `json:"line"`
	Func string `json:"func"`
}

func (f *FuncFrame) String() string { return stdfmt.Sprintf("%s %s:%d", f.Func, f.File, f.Line) }
func (f *FuncFrame) WriteTo(w stdio.Writer) (int64, error) {
	n, err := stdfmt.Fprintf(w, "%s %s:%d", f.Func, f.File, f.Line)
	return int64(n), err
}

// FrameLine — отрендеренный кадр стека одной строкой («func file:line»).
// Элемент блока результата StackBlocks / MarshalStackError; в JSON
// сериализуется как обычная строка.
type FrameLine string

// ParseStack разбирает одиночный кадр стека, отрендеренный одной строкой
// в формате «func file:line»: имя функции — до последнего пробела, файл —
// до последнего «:», номер строки — после него; номер должен быть ≥ 1.
// Разделитель между именем функции и «file:line» — последний пробел,
// поэтому имя функции не должно содержать пробелов. Возвращает ошибку,
// если строка не соответствует формату кадра: нет пробела, нет части
// «file:line», файл пуст, номер строки пуст, нечисловой или < 1.
// Результат предназначен для передачи в сторонние сервисы (маппинг на
// sentry.Frame: Func → Function, File → Filename, Line → LineNo).
func (f FrameLine) ParseStack() (*FuncFrame, error) {
	line := string(f)
	idx := stdstrings.LastIndex(line, " ")
	if idx <= 0 {
		return nil, stdfmt.Errorf("errors: ParseStack: неверный кадр «%s»", line)
	}
	fn := line[:idx]
	rest := line[idx+1:]
	colon := stdstrings.LastIndex(rest, ":")
	if colon <= 0 {
		return nil, stdfmt.Errorf("errors: ParseStack: неверный кадр «%s»", line)
	}
	ln, err := stdstrconv.Atoi(rest[colon+1:])
	if err != nil || ln < 1 {
		return nil, stdfmt.Errorf("errors: ParseStack: неверный кадр «%s»", line)
	}

	return &FuncFrame{Func: fn, File: rest[:colon], Line: ln}, nil
}

// MarshalStackError возвращает кадры стека ошибки в виде массива блоков
// кадров [][]FrameLine для записи в журнал: каждый блок — кадры одной
// ошибки цепочки или мульти-ошибки (строки «func file:line»). Сообщения
// ошибок в результат не попадают — они доступны через Error() и пишутся
// в поле error лога. Если err == nil, возвращает nil.
//
// Сигнатура `func(error) any` совместима с типичными marshaler-ами стека
// в JSON-логгерах. В JSON результат сериализуется как вложенный массив
// массивов строк.
func MarshalStackError(err error) any {
	if err == nil {
		return nil
	}

	return StackBlocks(err)
}

// StackBlocks строит блоки отрендеренных кадров стека ошибок: каждый
// элемент — кадры FrameLine (строки «func file:line») одной ошибки. Для
// цепочки (повторные Unwrap) — по блоку на каждую ошибку, реализующую
// StackTracer, в порядке от внешней ошибки к внутренним. Если в цепочке
// есть мульти-ошибка (тип с Unwrap() []error) — сначала блоки её
// под-ошибок (в порядке Unwrap() []error, рекурсивно: под-ошибка может
// сама быть мульти-ошибкой или цепочкой), затем блоки обёрток,
// окружающих этот узел мульти-ошибки, в порядке от внешней к внутренней.
// Кадры, для которых runtime.FuncForPC не находит функцию, пропускаются;
// ошибка без стека блока не формирует; если блоков нет вообще,
// возвращает nil.
func StackBlocks(err error) [][]FrameLine {
	var out, suffix [][]FrameLine
	for err != nil {
		if ms, ok := err.(interface{ Unwrap() []error }); ok {
			for _, e := range ms.Unwrap() {
				out = append(out, StackBlocks(e)...)
			}

			return append(out, suffix...)
		}
		if st, ok := err.(StackTracer); ok {
			block := make([]FrameLine, 0, len(st.StackTrace()))
			for _, f := range st.StackTrace() {
				if s := f.String(); s != "" {
					block = append(block, FrameLine(s))
				}
			}
			if len(block) > 0 {
				suffix = append(suffix, block)
			}
		}
		err = stderrors.Unwrap(err)
	}

	return suffix
}

var (
	StackSize = 10
	CutTail   = 2
	skipCall  = 3
)

// callers возвращает стек указателей на программу в точке вызова
// вызывающей функции.
func callers() Stack {
	pcs := make([]uintptr, StackSize)
	n := stdruntime.Callers(skipCall, pcs)
	if n > CutTail {
		n -= CutTail
	}
	pcs = pcs[:n]
	for i := range n {
		pcs[i]--
	}

	return *(*Stack)(stdunsafe.Pointer(&pcs))
}
