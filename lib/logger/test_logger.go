package logger

import (
	"bytes"
	"log"
	"net/url"
	"sync"

	"github.com/fatih/color"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type MemorySink struct {
	m sync.Mutex
	b bytes.Buffer
}

var _ zap.Sink = &MemorySink{}

func (s *MemorySink) Write(p []byte) (n int, err error) {
	s.m.Lock()
	defer s.m.Unlock()
	return s.b.Write(p)
}

func (s *MemorySink) Close() error { return nil }

func (s *MemorySink) Sync() error { return nil }

func (s *MemorySink) String() string {
	s.m.Lock()
	defer s.m.Unlock()
	return s.b.String()
}

var testMemoryLog *MemorySink
var createSinkOnce sync.Once

func registerMemorySink() {
	testMemoryLog = &MemorySink{m: sync.Mutex{}, b: bytes.Buffer{}}
	if err := zap.RegisterSink("memory", func(*url.URL) (zap.Sink, error) {
		return PrettyConsole{Sink: testMemoryLog}, nil
	}); err != nil {
		panic(err)
	}
}

func MemoryLogTestingOnly() *MemorySink {
	createSinkOnce.Do(registerMemorySink)
	return testMemoryLog
}

func CreateTestLogger(lvl zapcore.Level) *Logger {
	_ = MemoryLogTestingOnly() // Make sure memory log is created
	color.NoColor = false
	config := zap.NewProductionConfig()
	config.Level.SetLevel(lvl)
	config.OutputPaths = []string{"pretty://console", "memory://"}
	zl, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		log.Fatal(err)
	}
	return &Logger{
		SugaredLogger: zl.Sugar(),
	}
}

func CreateMemoryTestLogger(lvl zapcore.Level) *Logger {
	_ = MemoryLogTestingOnly() // Make sure memory log is created
	color.NoColor = true
	config := zap.NewProductionConfig()
	config.Level.SetLevel(lvl)
	config.OutputPaths = []string{"memory://"}
	zl, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		log.Fatal(err)
	}
	return &Logger{
		SugaredLogger: zl.Sugar(),
	}
}
