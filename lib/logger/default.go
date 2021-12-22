package logger

import (
	"errors"
	"fmt"
	"log"
	"os"

	"go.uber.org/zap"
)

var (
	Default *Logger
)

func init() {
	err := zap.RegisterSink("pretty", prettyConsoleSink(os.Stderr))
	if err != nil {
		log.Fatalf("failed to register pretty printer %+v", err)
	}
	err = registerOSSinks()
	if err != nil {
		log.Fatalf("failed to register os specific sinks %+v", err)
	}

	zl, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}
	SetLogger(&Logger{
		SugaredLogger: zl.Sugar(),
	})
}

func SetLogger(newLogger *Logger) {
	if Default != nil {
		defer func() {
			if err := Default.Sync(); err != nil {
				if errors.Unwrap(err).Error() != os.ErrInvalid.Error() &&
					errors.Unwrap(err).Error() != "inappropriate ioctl for device" &&
					errors.Unwrap(err).Error() != "bad file descriptor" {
					// logger.Sync() will return 'invalid argument' error when closing file
					log.Fatalf("failed to sync logger %+v", err)
				}
			}
		}()
	}
	Default = newLogger
}

func Infow(msg string, keysAndValues ...interface{}) {
	Default.Infow(msg, keysAndValues...)
}

func Debugw(msg string, keysAndValues ...interface{}) {
	Default.Debugw(msg, keysAndValues...)
}

func Warnw(msg string, keysAndValues ...interface{}) {
	Default.Warnw(msg, keysAndValues...)
}

func Errorw(msg string, keysAndValues ...interface{}) {
	Default.Errorw(msg, keysAndValues...)
}

func NewErrorw(msg string, keysAndValues ...interface{}) error {
	Default.Errorw(msg, keysAndValues...)
	return errors.New(msg)
}

func Infof(format string, values ...interface{}) {
	Default.Info(fmt.Sprintf(format, values...))
}

func Debugf(format string, values ...interface{}) {
	Default.Debug(fmt.Sprintf(format, values...))
}

func Tracef(format string, values ...interface{}) {
	Default.Debug("TRACE: " + fmt.Sprintf(format, values...))
}

func Warnf(format string, values ...interface{}) {
	Default.Warn(fmt.Sprintf(format, values...))
}

func Panicf(format string, values ...interface{}) {
	Default.Panic(fmt.Sprintf(format, values...))
}

func Info(args ...interface{}) {
	Default.Info(args...)
}

func Debug(args ...interface{}) {
	Default.Debug(args...)
}

func Trace(args ...interface{}) {
	Default.Debug(append([]interface{}{"TRACE: "}, args...))
}

func Warn(args ...interface{}) {
	Default.Warn(args...)
}

func Error(args ...interface{}) {
	Default.Error(args...)
}

func WarnIf(err error) {
	Default.WarnIf(err)
}

func ErrorIf(err error, optionalMsg ...string) {
	Default.ErrorIf(err, optionalMsg...)
}

func ErrorIfCalling(f func() error, optionalMsg ...string) {
	Default.ErrorIfCalling(f, optionalMsg...)
}

func Fatal(args ...interface{}) {
	Default.Fatal(args...)
}

func Errorf(format string, values ...interface{}) {
	Error(fmt.Sprintf(format, values...))
}

func Fatalf(format string, values ...interface{}) {
	Fatal(fmt.Sprintf(format, values...))
}

func Fatalw(msg string, keysAndValues ...interface{}) {
	Default.Fatalw(msg, keysAndValues...)
}

func Panic(args ...interface{}) {
	Default.Panic(args...)
}

func PanicIf(err error) {
	Default.PanicIf(err)
}

func Sync() error {
	return Default.Sync()
}
