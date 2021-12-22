package logger

import (
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

var levelColors = map[string]func(...interface{}) string{
	"default": color.New(color.FgWhite).SprintFunc(),
	"debug":   color.New(color.FgGreen).SprintFunc(),
	"info":    color.New(color.FgWhite).SprintFunc(),
	"warn":    color.New(color.FgYellow).SprintFunc(),
	"error":   color.New(color.FgRed).SprintFunc(),
	"panic":   color.New(color.FgRed).SprintFunc(),
	"fatal":   color.New(color.FgRed).SprintFunc(),
}

var blue = color.New(color.FgBlue).SprintFunc()
var green = color.New(color.FgGreen).SprintFunc()


type PrettyConsole struct {
	zap.Sink
}


func (pc PrettyConsole) Write(b []byte) (int, error) {
	if !gjson.ValidBytes(b) {
		return 0, fmt.Errorf("unable to parse json for pretty console: %s", string(b))
	}
	js := gjson.ParseBytes(b)
	headline := generateHeadline(js)
	details := generateDetails(js)
	return pc.Sink.Write([]byte(fmt.Sprintln(headline, details)))
}

func generateHeadline(js gjson.Result) string {
	sec, dec := math.Modf(js.Get("ts").Float())
	headline := []interface{}{
		iso8601UTC(time.Unix(int64(sec), int64(dec*(1e9)))),
		" ",
		coloredLevel(js.Get("level")),
		fmt.Sprintf("%-50s", js.Get("msg")),
		" ",
		fmt.Sprintf("%-32s", blue(js.Get("caller"))),
	}
	return fmt.Sprint(headline...)
}

var detailsBlacklist = map[string]bool{
	"level":  true,
	"ts":     true,
	"msg":    true,
	"caller": true,
	"hash":   true,
}

func generateDetails(js gjson.Result) string {
	data := js.Map()
	keys := []string{}

	for k := range data {
		if detailsBlacklist[k] || len(data[k].String()) == 0 {
			continue
		}
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var details strings.Builder

	for _, v := range keys {
		details.WriteString(fmt.Sprintf("%s=%v ", green(v), data[v]))
	}

	return details.String()
}

func coloredLevel(level gjson.Result) string {
	color, ok := levelColors[level.String()]
	if !ok {
		color = levelColors["default"]
	}
	return color(fmt.Sprintf("%-8s", fmt.Sprint("[", strings.ToUpper(level.String()), "]")))
}

func iso8601UTC(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func prettyConsoleSink(s zap.Sink) func(*url.URL) (zap.Sink, error) {
	return func(*url.URL) (zap.Sink, error) {
		return PrettyConsole{s}, nil
	}
}
