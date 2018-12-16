package logger

import "go.uber.org/zap"

var logger *zap.Logger

func InitLogger(cfg zap.Config) {
	var err error
	logger, err = cfg.Build()
	if err != nil {
		panic(err)
	}
}

func Info(msg string, field ...zap.Field) {
	logger.Info(msg, field...)
}

func Debug(msg string, field ...zap.Field) {
	logger.Debug(msg, field...)
}

func Error(msg string, field ...zap.Field) {
	logger.Error(msg, field...)
}

func Warn(msg string, field ...zap.Field) {
	logger.Warn(msg, field...)
}

func DPanic(msg string, field ...zap.Field) {
	logger.DPanic(msg, field...)
}

func Panic(msg string, field ...zap.Field) {
	logger.Panic(msg, field...)
}

func Fatal(msg string, field ...zap.Field) {
	logger.Fatal(msg, field...)
}

func Sync() {
	logger.Sync()
}

