package logger

import (
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"path"
)

var logger *Logger

type Logger struct {
	*zap.SugaredLogger
}

const log_dir = "./phoenix"
func init() {
	writeSyncer := getLogWriter(log_dir)
	encoder := getEncoder()
	core := zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel)

	log := zap.New(core,zap.AddCaller())
	logger = &Logger{log.Sugar()}
}

func getEncoder() zapcore.Encoder {
	return zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
}

func getLogWriter(dir string) zapcore.WriteSyncer {
	destination := path.Join(dir, "test.log")
	lumberJackLogger := &lumberjack.Logger{
		Filename:   destination,
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   false,
	}

	return zapcore.AddSync(lumberJackLogger)
}

func NewLogger(dir string) *Logger {
	writeSyncer := getLogWriter(dir)
	encoder := getEncoder()
	core := zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel)

	log := zap.New(core,zap.AddCaller())
	logger = &Logger{log.Sugar()}
	return logger
}

func SetLoggerDir(dir string) {
	defer logger.Sync()
	logger = NewLogger(dir)
}

func (self *Logger) Write(b []byte) (n int, err error) {
	self.Info(string(b))
	return len(b), nil
}

func GetLogger() *Logger {
	return logger
}

func SetLogger(newLogger *Logger) {
	defer logger.Sync()
	logger = newLogger
}


func LoggerWriter() *Logger {
	writeSyncer := getLogWriter("")
	encoder := getEncoder()
	core := zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel)

	log := zap.New(core/*,zap.AddCaller()*/)
	logger = &Logger{log.Sugar()}
	return logger
}

func Errorw(msg string, keysAndValues ...interface{}) {
	logger.Errorw(msg, keysAndValues...)
}

func Infow(msg string, keysAndValues ...interface{}) {
	logger.Infow(msg, keysAndValues...)
}

func Info(args ...interface{}) {
	logger.Info(args)
}

func Fatal(args ...interface{}) {
	logger.Fatal(args)
}

func Panic(args ...interface{}) {
	logger.Panic(args)
}

func Sync() error {
	return logger.Sync()
}
