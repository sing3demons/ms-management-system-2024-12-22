package ms

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Ensure the /log/detail directory exists

// Function to generate log file name based on appName and timestamp (pattern: appName_YYYYMMDD_HHmmss.log)
func getLogFileName(t time.Time) string {
	appName := os.Getenv("SERVICE_NAME")
	if appName == "" {
		appName = "go-service"
	}
	year, month, day := t.Date()
	hour, minute, second := t.Clock()

	return fmt.Sprintf("%s_%04d%02d%02d_%02d%02d%02d.log", appName, year, month, day, hour, minute, second)
}

func NewLogger(wf AppLog) *zap.Logger {
	// create a zapcore encoder config
	var encCfg zapcore.EncoderConfig
	encCfg.MessageKey = "msg"

	core := zapcore.NewCore(zapcore.NewConsoleEncoder(encCfg), zapcore.AddSync(os.Stdout), zap.InfoLevel)

	if wf.LogFile {
		logFile := filepath.Join(wf.Name, getLogFileName(time.Now()))
		if _, err := os.Stat(wf.Name); os.IsNotExist(err) {
			err := os.MkdirAll(wf.Name, os.ModePerm)
			if err != nil {
				panic("failed to create log directory")
			}
		}

		w := zapcore.NewMultiWriteSyncer(zapcore.AddSync(&lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    10, // megabytes
			MaxBackups: 3,
			MaxAge:     7,
		}), zapcore.AddSync(os.Stdout))

		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(encCfg),
			w,
			zap.InfoLevel,
		)
	}

	return zap.New(core)
}

func createLogger(path string) (*zap.Logger, error) {
	// Create log file with rotating mechanism
	logFile := filepath.Join(path, getLogFileName(time.Now()))

	// Create a zapcore encoder config
	encCfg := zapcore.EncoderConfig{
		MessageKey:   "msg",
		TimeKey:      "time",
		LevelKey:     "level",
		CallerKey:    "caller",
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	// File encoder using console format
	fileEncoder := zapcore.NewConsoleEncoder(encCfg)

	// Setting up lumberjack logger for log rotation
	writerSync := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    500, // megabytes
		MaxBackups: 3,   // number of backups
		MaxAge:     1,   // days
		LocalTime:  true,
		Compress:   true, // compress the backups
	})

	// Create the core with InfoLevel logging
	core := zapcore.NewCore(fileEncoder, writerSync, zap.InfoLevel)

	// Create logger
	log := zap.New(core)

	return log, nil
}

func NewLogFile(path string) *zap.Logger {
	log, err := createLogger(path)
	if err != nil {
		fmt.Println("Failed to create log file logger:", err)
	}
	return log
}
