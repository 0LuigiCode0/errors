# errors

Пакет ошибок для Go: добавление контекста, трассировка стека, числовые коды
ошибок и мульти-ошибки.

Модуль: `github.com/0LuigiCode0/errors` (Go 1.26+)

```bash
go get github.com/0LuigiCode0/errors
```

## Зачем

Традиционный идиоматический способ обработки ошибок в Go:

```go
if err != nil {
    return err
}
```

При рекурсивном применении вверх по стеку вызовов это приводит к сообщениям
об ошибках без контекста и отладочной информации. Пакет `errors` позволяет
добавлять контекст на путь ошибки, не разрушая исходное значение ошибки.

## Создание ошибок и добавление контекста

| Функция | Что делает |
|---|---|
| `New(msg)` / `Newf(fmt, ...)` | новая ошибка с сообщением и трассировкой стека в точке вызова |
| `Wrap(err, msg)` / `Wrapf(err, fmt, ...)` | обёртка: трассировка стека + сообщение |
| `WithStack(err)` | только трассировка стека |
| `WithMessage(err, msg)` / `WithMessagef(err, fmt, ...)` | только сообщение (без стека) |

Все функции, кроме `New`/`Newf`, возвращают `nil`, если `err == nil`.

```go
_, err := io.ReadAll(r)
if err != nil {
    return errors.Wrap(err, "read failed")
}
```

`Wrap` эквивалентен композиции `WithStack(WithMessage(err, msg))`; текст
ошибки — цепочка сообщений всех уровней: `"read failed: <внутренняя ошибка>"`.

## Коды ошибок

Каждая ошибка пакета несёт числовой код — метод `Code() int`. У ошибки,
созданной без явного кода, код равен `NaNCode` (-1).

Конструкторы с суффиксом `Code` — точные аналоги базовых функций, которые
помечают результат заданным кодом:

| Функция | Аналог |
|---|---|
| `NewCode(code, msg)` / `NewCodef(code, fmt, ...)` | `New` / `Newf` с кодом |
| `WithStackCode(code, err)` | `WithStack` с кодом |
| `WithMessageCode(code, err, msg)` / `WithMessageCodef(...)` | `WithMessage` / `WithMessagef` с кодом |
| `WrapCode(code, err, msg)` / `WrapCodef(code, err, fmt, ...)` | `Wrap` / `Wrapf` с кодом на обёртке |
| `WrapMultiCode(code, errs, msg)` / `WrapMultiCodef(...)` | `WrapMulti` / `WrapMultif` с кодом на обёртке |

Поиск кода по цепочке (включая мульти-ошибки):

```go
err := errors.WrapCode(1001, errors.NewCode(1002, "base"), "context")
errors.IsCode(err, 1001)          // true
errors.IsCode(err, 1002)          // true
errors.IsCode(err, 1001, 9999)    // true — коды принимаются вариативно
```

`IsCode` обходит цепочку `Unwrap() error` и `Unwrap() []error` и возвращает
true, если `Code()` любой ошибки из неё входит в переданный список. Для
`nil`-ошибки или пустого списка кодов возвращает false.

## Мульти-ошибки

`WrapMulti` объединяет несколько ошибок стандартным `errors.Join` и помечает
их трассировкой стека и сообщением. Ошибки, равные nil, пропускаются; если
не-nil ошибок нет — возвращается nil.

```go
err := errors.WrapMulti([]error{err1, err2, nil}, "failed to process")
```

Форматированные и кодированные версии: `WrapMultif`, `WrapMultiCode`,
`WrapMultiCodef`.

Вывод мульти-ошибки через `%+v` нумерует вложенные ошибки:

```
failed to process:
1. err1
2. err2
<кадры стека>
```

Мульти-ошибка строится на стандартном `errors.Join`, поэтому поддерживает
`Unwrap() []error`: `errors.Is` и `errors.As` проходят в под-ошибки. В журнале
(через `MarshalStackError`) поле `stack` мульти-ошибки — вложенный массив
блоков кадров: блоки под-ошибок в порядке `Join`, затем блок обёртки
`WrapMulti`; сообщения под-ошибок — только в поле `error`.

## Проверка ошибок

| Функция | Назначение |
|---|---|
| `Is(err, target)` | есть ли `target` в цепочке (стандартная семантика Go 1.13) |
| `Iss(err, targets...)` | как `Is`, но с любым числом целей; обходит также `Unwrap() []error` |
| `As(err, target)` | извлечение типа из цепочки (стандартная семантика) |
| `Unwrap(err)` | развёртка одного шага цепочки |
| `IsCode(err, codes...)` | совпадает ли `Code()` любой ошибки в цепочке с одним из `codes` |
| `Cause(err)` | первопричина — самая глубокая ошибка, не реализующая `Cause() error` |

Использование `Wrap` строит стек ошибок; `Cause` разворачивает его до
исходной:

```go
switch errors.Cause(err).(type) {
case *MyError:
    // обработать конкретно
default:
    // неизвестная ошибка
}
```

Интерфейс `causer` не экспортируется, но считается частью стабильного
публичного интерфейса пакета.

## Форматированный вывод

Все значения ошибок пакета реализуют `fmt.Formatter`:

| Верба | Результат |
|---|---|
| `%s` / `%v` | полное сообщение: цепочка «msg: ... : cause» |
| `%q` | то же, в кавычках |
| `%+v` | сообщение + рекурсивный вывод всех кадров трассировки стека |

## JSON-сериализация ошибок

`json.Marshal` ошибки пакета сериализует её как JSON-строку с текстом
ошибки (`Error()`), а не внутреннее тело структуры:

```go
json.Marshal(errors.Wrap(errors.New("whoops"), "read failed"))
// → "read failed: whoops"
```

Трассировка стека в эту строку не попадает — для лога её выводит
`MarshalStackError` (поле `stack`).

## Трассировка стека

`New*`, `WithStack*` и все `Wrap*` записывают трассировку стека в точке
вызова. Получить её можно через интерфейс `StackTracer`:

```go
type StackTracer interface {
    StackTrace() Stack
}

if st, ok := err.(errors.StackTracer); ok {
    fmt.Println(st.StackTrace()) // Stack = []FuncPtr
}
```

`Stack` (`type Stack []FuncPtr`) сам реализует `StackTracer` и `Stringer`:
каждый кадр печатается одной строкой в формате `func file:line`.

Кадры представляются типом `FrameLine` — отрендеренный кадр стека одной
строкой (`func file:line`); в JSON сериализуется как обычная строка:

```go
type FrameLine string

func (f FrameLine) ParseStack() (*FuncFrame, error)
```

`StackBlocks(err) [][]FrameLine` — блоки кадров стека ошибок: каждый
элемент — кадры (строки `func file:line`) одной ошибки. Для цепочки
(повторные `Unwrap`) — по блоку на каждую ошибку, реализующую `StackTracer`,
в порядке от внешней ошибки к внутренним; кадры внутри блока идут сверху
вниз. Для мульти-ошибки (в цепочке есть тип с `Unwrap() []error`) — сначала
блоки под-ошибок в порядке `Unwrap() []error` (рекурсивно), затем блоки
обёрток, окружающих этот узел, от внешней к внутренней. Ошибка без стека
блока не формирует; если блоков нет, возвращает `nil`:

```go
blocks := errors.StackBlocks(err) // [][]FrameLine, каждый блок — кадры одной ошибки
```

`MarshalStackError(err) any` — то же самое, готовое к записи в журнал:
массив блоков кадров `[][]FrameLine` (или `nil` для `nil`-ошибки); в JSON —
вложенный массив массивов строк. Сообщения об ошибках в результат не
попадают — они в `Error()` и в поле `error` лога.

`FrameLine.ParseStack() (*FuncFrame, error)` — разбирает одиночный кадр,
отрендеренный одной строкой, в `FuncFrame`, чтобы передать его в сторонние
сервисы. Каждый кадр описывается `FuncFrame`:

```go
type FuncFrame struct {
    File string `json:"source"`
    Line int    `json:"line"`
    Func string `json:"func"`
}

frame, err := FrameLine(frameLine).ParseStack() // frameLine — строка «func file:line»
// Func → sentry.Frame.Function, File → Filename, Line → LineNo
```

Глубина захвата стека настраивается переменными `StackSize` (по умолчанию 10)
и `CutTail` (по умолчанию 2). `CutTail` кадров срезается с конца захвата;
если глубина захвата ≤ `CutTail`, обрезка не выполняется (стек не срезается
до нуля).

## Интеграция с zerolog

Стек ошибок можно подключить к логу `zerolog`:

```go
func init() {
    zerolog.ErrorStackMarshaler = errors.MarshalStackError
}

log := zerolog.New(os.Stderr)
log.Err(err).Stack().Msg("failed")
```

Поле `stack` в JSON-логе — вложенный массив блоков кадров: каждый
внутренний массив — однострочные кадры `func file:line` одной ошибки
цепочки, без `\n`/`\t` внутри элементов (в raw-JSON экспайзы отсутствуют).
Для мульти-ошибки — сначала блоки под-ошибок, затем блок обёртки:

```json
{
  "level": "error",
  "error": "read failed: whoops",
  "message": "failed",
  "stack": [
    [
      "gitlab.innoseti.ru/.../errors_test.TestZerologErrorStackMarshaler /home/user/.../zerolog_test.go:33",
      "testing.tRunner /usr/local/go/.../testing/testing.go:2036",
      "runtime.goexit /usr/local/go/.../runtime/asm_amd64.s:1771"
    ],
    [
      "gitlab.innoseti.ru/.../errors_test.TestZerologErrorStackMarshaler /home/user/.../zerolog_test.go:31",
      "testing.tRunner /usr/local/go/.../testing/testing.go:2036",
      "runtime.goexit /usr/local/go/.../runtime/asm_amd64.s:1771"
    ]
  ]
}
```

— см. тест `TestZerologErrorStackMarshaler` (`zerolog_test.go`).

### Передача стека в Sentry

Поле `stack` в логе — это массив блоков кадров `[][]FrameLine`: каждый блок
— кадры `func file:line` одной ошибки. Чтобы передать кадры в Sentry,
пройдите по блокам и кадрам, каждый кадр разберите `fl.ParseStack()`
(однострочный кадр раскладывается по последнему пробелу и последнему «:»)
и преобразуйте `FuncFrame` в `sentry.Frame` (импорт sentry в репо не нужен):

```go
for _, block := range stack { // stack — [][]FrameLine, поле "stack" из лога
    for _, fl := range block {
        frame, err := fl.ParseStack()
        if err != nil {
            continue
        }
        // sentry.Frame{Function: frame.Func, Filename: frame.File, LineNo: int64(frame.Line)}
    }
}
```

[Документация пакета](https://github.com/0LuigiCode0/errors).

## License

BSD-2-Clause