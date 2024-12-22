package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func NewLog(file, console bool) *zap.Logger {
	if file {
		return NewLogFile(console)
	}
	return NewLogger()
}

func NewLogger() *zap.Logger {
	var encCfg zapcore.EncoderConfig
	encCfg.MessageKey = "msg"

	w := zapcore.AddSync(os.Stdout)
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(encCfg), w, zap.InfoLevel)

	log := zap.New(core)

	return log
}

func NewLogFile(console bool) *zap.Logger {
	// Create log file with rotating mechanism
	const logDir = "./logs/app"
	logFile := filepath.Join(logDir, getLogFileName(time.Now()))
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		err := os.MkdirAll(logDir, os.ModePerm)
		if err != nil {
			panic("failed to create log directory")
		}
	}

	var encCfg zapcore.EncoderConfig
	encCfg.MessageKey = "msg"

	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     7,
	})
	if console {
		w = zapcore.NewMultiWriteSyncer(w, zapcore.AddSync(os.Stdout))
	}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encCfg),
		w,
		zap.InfoLevel,
	)
	return zap.New(core)
}

func getLogFileName(t time.Time) string {
	appName := os.Getenv("SERVICE_NAME")
	if appName == "" {
		appName = "go-service"
	}
	year, month, day := t.Date()
	hour, minute, second := t.Clock()

	return fmt.Sprintf("%s_%04d%02d%02d_%02d%02d%02d.log", appName, year, month, day, hour, minute, second)
}
