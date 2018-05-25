package horizontal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/nwidger/jsoncolor"
	"github.com/olekukonko/ts"
	"github.com/rs/zerolog"
)

var (
	f          = jsoncolor.NewFormatter()
	size       int
	horizontal []byte
)

func init() {
	// json colors
	f.SpaceColor = color.New(color.FgRed, color.Bold)
	f.CommaColor = color.New(color.FgWhite, color.Bold)
	f.ColonColor = color.New(color.FgYellow, color.Bold)
	f.ObjectColor = color.New(color.FgGreen, color.Bold)
	f.ArrayColor = color.New(color.FgHiRed)
	f.FieldColor = color.New(color.FgCyan)
	f.StringColor = color.New(color.FgHiYellow)
	f.TrueColor = color.New(color.FgCyan, color.Bold)
	f.FalseColor = color.New(color.FgHiRed)
	f.NumberColor = color.New(color.FgHiMagenta)
	f.NullColor = color.New(color.FgWhite, color.Bold)
	f.StringQuoteColor = color.New(color.FgBlue, color.Bold)

	// terminal size
	s, _ := ts.GetSize()
	char := byte('=')
	size = s.Col()
	for i := 0; i < size; i++ {
		horizontal = append(horizontal, char)
	}
}

const (
	cReset    = 0
	cBold     = 1
	cRed      = 31
	cGreen    = 32
	cYellow   = 33
	cBlue     = 34
	cMagenta  = 35
	cCyan     = 36
	cGray     = 37
	cDarkGray = 90
)

var consoleBufPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 100))
	},
}

type ConsoleWriter struct {
	Out     io.Writer
	NoColor bool
}

func (w ConsoleWriter) Write(p []byte) (n int, err error) {
	var event map[string]interface{}
	// p = decodeIfBinaryToBytes(p)
	d := json.NewDecoder(bytes.NewReader(p))
	d.UseNumber()
	err = d.Decode(&event)
	if err != nil {
		return
	}
	buf := consoleBufPool.Get().(*bytes.Buffer)
	defer consoleBufPool.Put(buf)
	buf.Write(horizontal)
	lvlColor := cReset
	level := "????"
	if l, ok := event[zerolog.LevelFieldName].(string); ok {
		if !w.NoColor {
			lvlColor = levelColor(l)
		}
		level = strings.ToUpper(l)[0:4]
	}
	fmt.Fprintf(buf, "%s |%s| %s",
		colorize(formatTime(event[zerolog.TimestampFieldName]), cDarkGray, !w.NoColor),
		colorize(level, lvlColor, !w.NoColor),
		colorize(event[zerolog.MessageFieldName], cReset, !w.NoColor))
	fields := make([]string, 0, len(event))
	for field := range event {
		switch field {
		case zerolog.LevelFieldName, zerolog.TimestampFieldName, zerolog.MessageFieldName:
			continue
		}
		fields = append(fields, field)
		buf.WriteByte('\n')
	}
	sort.Strings(fields)
	for _, field := range fields {
		fmt.Fprintf(buf, " %s=", colorize(field, cCyan, !w.NoColor))
		switch value := event[field].(type) {
		case string:
			if needsQuote(value) {
				buf.WriteString(strconv.Quote(value))
			} else {
				buf.WriteString(value)
			}
		case json.Number:
			fmt.Fprint(buf, value)
		default:
			v, err := json.Marshal(value)
			if err != nil {
				return 0, err
			}

			err = f.Format(buf, v)
			if err != nil {
				return 0, err
			}
		}
		buf.WriteByte('\n')
	}
	buf.WriteByte('\n')
	buf.WriteTo(w.Out)
	n = len(p)
	return
}

func formatTime(t interface{}) string {
	switch t := t.(type) {
	case string:
		return t
	case json.Number:
		u, _ := t.Int64()
		return time.Unix(u, 0).Format(time.RFC3339)
	}
	return "<nil>"
}

func colorize(s interface{}, color int, enabled bool) string {
	if !enabled {
		return fmt.Sprintf("%v", s)
	}
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", color, s)
}

func levelColor(level string) int {
	switch level {
	case "debug":
		return cMagenta
	case "info":
		return cGreen
	case "warn":
		return cYellow
	case "error", "fatal", "panic":
		return cRed
	default:
		return cReset
	}
}

func needsQuote(s string) bool {
	for i := range s {
		if s[i] < 0x20 || s[i] > 0x7e || s[i] == ' ' || s[i] == '\\' || s[i] == '"' {
			return true
		}
	}
	return false
}
