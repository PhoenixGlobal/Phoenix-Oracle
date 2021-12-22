package logger

import (
	"log"
	"reflect"
	"runtime"

	"gorm.io/gorm"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.SugaredLogger
	Orm         ORM
	lvl         zapcore.Level
	dir         string
	jsonConsole bool
	toDisk      bool
}

const (
	HeadTracker = "head_tracker"
	FluxMonitor = "fluxmonitor"
	Keeper      = "keeper"
)

func GetLogServices() []string {
	return []string{
		HeadTracker,
		FluxMonitor,
		Keeper,
	}
}

func (l *Logger) Write(b []byte) (int, error) {
	l.Info(string(b))
	return len(b), nil
}

func (l *Logger) With(args ...interface{}) *Logger {
	newLogger := CreateLogger(l.SugaredLogger.With(args...))
	newLogger.Orm = l.Orm
	newLogger.lvl = l.lvl
	newLogger.dir = l.dir
	newLogger.jsonConsole = l.jsonConsole
	newLogger.toDisk = l.toDisk
	return newLogger
}

func (l *Logger) Named(name string) *Logger {
	newLogger := CreateLogger(l.SugaredLogger.Named(name).With("id", name))
	newLogger.Orm = l.Orm
	newLogger.lvl = l.lvl
	newLogger.dir = l.dir
	newLogger.jsonConsole = l.jsonConsole
	newLogger.toDisk = l.toDisk
	return newLogger
}

func (l *Logger) WithError(err error) *Logger {
	return l.With("error", err)
}

func (l *Logger) WarnIf(err error) {
	if err != nil {
		l.Warn(err)
	}
}

func (l *Logger) ErrorIf(err error, optionalMsg ...string) {
	if err != nil {
		if len(optionalMsg) > 0 {
			l.Error(errors.Wrap(err, optionalMsg[0]))
		} else {
			l.Error(err)
		}
	}
}

func (l *Logger) ErrorIfCalling(f func() error, optionalMsg ...string) {
	err := f()
	if err != nil {
		e := errors.Wrap(err, runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name())
		if len(optionalMsg) > 0 {
			Default.Error(errors.Wrap(e, optionalMsg[0]))
		} else {
			Default.Error(e)
		}
	}
}

func (l *Logger) PanicIf(err error) {
	if err != nil {
		l.Panic(err)
	}
}

func (l *Logger) SetDB(db *gorm.DB) {
	l.Orm = NewORM(db)
}

func (l *Logger) GetServiceLogLevels() (map[string]string, error) {
	serviceLogLevels := make(map[string]string)

	for _, svcName := range GetLogServices() {
		svc, err := l.ServiceLogLevel(svcName)
		if err != nil {
			Fatalf("error getting service log levels: %v", err)
		}
		serviceLogLevels[svcName] = svc
	}

	return serviceLogLevels, nil
}

func CreateLogger(zl *zap.SugaredLogger) *Logger {
	return &Logger{
		SugaredLogger: zl,
	}
}

func CreateLoggerWithConfig(zl *zap.SugaredLogger, lvl zapcore.Level, dir string, jsonConsole bool, toDisk bool) *Logger {
	return &Logger{
		SugaredLogger: zl,
		lvl:           lvl,
		dir:           dir,
		jsonConsole:   jsonConsole,
		toDisk:        toDisk,
	}
}

func initLogConfig(dir string, jsonConsole bool, lvl zapcore.Level, toDisk bool) zap.Config {
	config := zap.NewProductionConfig()
	if !jsonConsole {
		config.OutputPaths = []string{"pretty://console"}
	}
	if toDisk {
		destination := logFileURI(dir)
		config.OutputPaths = append(config.OutputPaths, destination)
		config.ErrorOutputPaths = append(config.ErrorOutputPaths, destination)
	}
	config.Level.SetLevel(lvl)
	return config
}

func CreateProductionLogger(
	dir string, jsonConsole bool, lvl zapcore.Level, toDisk bool) *Logger {
	config := initLogConfig(dir, jsonConsole, lvl, toDisk)

	zl, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		log.Fatal(err)
	}
	return CreateLoggerWithConfig(zl.Sugar(), lvl, dir, jsonConsole, toDisk)
}

func (l *Logger) InitServiceLevelLogger(serviceName string, logLevel string) (*Logger, error) {
	var ll zapcore.Level
	if err := ll.UnmarshalText([]byte(logLevel)); err != nil {
		return nil, err
	}

	config := initLogConfig(l.dir, l.jsonConsole, ll, l.toDisk)

	zl, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, err
	}

	return CreateLoggerWithConfig(zl.Named(serviceName).Sugar(), ll, l.dir, l.jsonConsole, l.toDisk), nil
}

func (l *Logger) ServiceLogLevel(serviceName string) (string, error) {
	if l.Orm != nil {
		level, err := l.Orm.GetServiceLogLevel(serviceName)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			Warnf("Error while trying to fetch %s service log level: %v", serviceName, err)
		} else if err == nil {
			return level, nil
		}
	}
	return l.lvl.String(), nil
}

func NewProductionConfig(lvl zapcore.Level, dir string, jsonConsole, toDisk bool) (c zap.Config) {
	var outputPath string
	if jsonConsole {
		outputPath = "stderr"
	} else {
		outputPath = "pretty://console"
	}
	// Mostly copied from zap.NewProductionConfig with sampling disabled
	c = zap.Config{
		Level:            zap.NewAtomicLevelAt(lvl),
		Development:      false,
		Sampling:         nil,
		Encoding:         "json",
		EncoderConfig:    NewProductionEncoderConfig(),
		OutputPaths:      []string{outputPath},
		ErrorOutputPaths: []string{"stderr"},
	}
	if toDisk {
		destination := logFileURI(dir)
		c.OutputPaths = append(c.OutputPaths, destination)
		c.ErrorOutputPaths = append(c.ErrorOutputPaths, destination)
	}
	return
}

func NewProductionEncoderConfig() zapcore.EncoderConfig {
	// Copied from zap.NewProductionEncoderConfig but with ISO timestamps instead of Unix
	return zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}
