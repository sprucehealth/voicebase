package golog

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
	"unicode/utf8"
)

const (
	timeFormat  = "2006-01-02T15:04:05-0700"
	floatFormat = 'f'
)

type Formatter interface {
	Format(e *Entry) []byte
}

type FormatterFunc func(*Entry) []byte

func (f FormatterFunc) Format(e *Entry) []byte {
	return f(e)
}

func JSONFormatter() Formatter {
	return FormatterFunc(func(e *Entry) []byte {
		js := make(map[string]interface{}, len(e.Ctx)/2+4)
		for i := 0; i < len(e.Ctx); i += 2 {
			k, ok := e.Ctx[i].(string)
			if !ok {
				js["_error"] = fmt.Sprintf("%+v is not a string key", k)
			} else {
				js[k] = e.Ctx[i+1]
			}
		}
		js["t"] = e.Time.Format(timeFormat)
		js["level"] = e.Lvl.String()
		js["msg"] = e.Msg
		if e.Src != "" {
			js["src"] = e.Src
		}

		b, err := json.Marshal(js)
		if err != nil {
			b, _ = json.Marshal(map[string]string{"JSONFormatterError": err.Error()})
			return b
		}
		return b
	})
}

func LogfmtFormatter() Formatter {
	return FormatterFunc(func(e *Entry) []byte {
		buf := &bytes.Buffer{}
		buf.WriteString("t=")
		buf.WriteString(e.Time.Format(timeFormat))
		buf.WriteString(" lvl=")
		buf.WriteString(e.Lvl.String())
		buf.WriteString(" msg=")
		buf.WriteString(quoteASCII(e.Msg))
		if e.Src != "" {
			buf.WriteString(" src=")
			buf.WriteString(quoteASCII(e.Src))
		}
		for i := 0; i < len(e.Ctx); i += 2 {
			k, ok := e.Ctx[i].(string)
			if !ok {
				buf.WriteString(" _error=")
			} else {
				buf.WriteByte(' ')
				buf.WriteString(k)
				buf.WriteByte('=')
			}
			buf.WriteString(format(e.Ctx[i+1]))
		}
		buf.WriteByte('\n')
		return buf.Bytes()
	})
}

func FormatContext(ctx []interface{}, delim rune) []byte {
	buf := &bytes.Buffer{}
	for i := 0; i < len(ctx); i += 2 {
		if i != 0 {
			buf.WriteRune(delim)
		}
		k, ok := ctx[i].(string)
		if !ok {
			buf.WriteString("_error=")
		} else {
			buf.WriteString(k)
			buf.WriteByte('=')
		}
		buf.WriteString(format(ctx[i+1]))
	}
	return buf.Bytes()
}

func format(value interface{}) string {
	if value == nil {
		return "nil"
	}

	switch v := value.(type) {
	case bool:
		return strconv.FormatBool(v)
	case float32:
		return strconv.FormatFloat(float64(v), floatFormat, 3, 64)
	case float64:
		return strconv.FormatFloat(v, floatFormat, 3, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case int8, int16, int32, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", value)
	case string:
		return quoteASCII(v)
	case time.Time:
		return v.Format(timeFormat)
	case error:
		return quoteASCII(v.Error())
	case fmt.Stringer:
		return quoteASCII(v.String())
	case encoding.TextMarshaler:
		b, err := v.MarshalText()
		if err != nil {
			return quoteASCII("logFormatError:" + err.Error())
		}
		return quoteASCII(string(b))
	default:
		return quoteASCII(fmt.Sprintf("%+v", value))
	}
}

const lowerhex = "0123456789abcdef"

func quoteASCII(s string) string {
	if len(s) == 0 {
		return s
	}
	needsQuotes := false
	buf := make([]byte, 0, 3*len(s)/2) // Try to avoid more allocations.
	buf = append(buf, '"')
	for width := 0; len(s) > 0; s = s[width:] {
		r := rune(s[0])
		width = 1
		if r >= utf8.RuneSelf {
			r, width = utf8.DecodeRuneInString(s)
		}
		if width == 1 && r == utf8.RuneError {
			buf = append(buf, `\x`...)
			buf = append(buf, lowerhex[s[0]>>4])
			buf = append(buf, lowerhex[s[0]&0xF])
			continue
		}
		switch r {
		case '"', '=', ' ':
			needsQuotes = true
		}
		if r == rune('"') || r == '\\' { // always backslashed
			buf = append(buf, '\\')
			buf = append(buf, byte(r))
			continue
		}
		if r < utf8.RuneSelf && strconv.IsPrint(r) {
			buf = append(buf, byte(r))
			continue
		}
		switch r {
		case '\a':
			buf = append(buf, `\a`...)
		case '\b':
			buf = append(buf, `\b`...)
		case '\f':
			buf = append(buf, `\f`...)
		case '\n':
			buf = append(buf, `\n`...)
		case '\r':
			buf = append(buf, `\r`...)
		case '\t':
			buf = append(buf, `\t`...)
		case '\v':
			buf = append(buf, `\v`...)
		default:
			switch {
			case r < ' ':
				buf = append(buf, `\x`...)
				buf = append(buf, lowerhex[s[0]>>4])
				buf = append(buf, lowerhex[s[0]&0xF])
			case r > utf8.MaxRune:
				r = 0xFFFD
				fallthrough
			case r < 0x10000:
				buf = append(buf, `\u`...)
				for s := 12; s >= 0; s -= 4 {
					buf = append(buf, lowerhex[r>>uint(s)&0xF])
				}
			default:
				buf = append(buf, `\U`...)
				for s := 28; s >= 0; s -= 4 {
					buf = append(buf, lowerhex[r>>uint(s)&0xF])
				}
			}
		}
	}
	buf = append(buf, '"')
	if !needsQuotes {
		buf = buf[1 : len(buf)-1]
	}
	return string(buf)

}
