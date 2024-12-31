package main

import (
	"os"

	"github.com/sing3demons/logger-kp/logger"
	"github.com/sing3demons/saram-kafka/handler"
	"github.com/sing3demons/saram-kafka/microservice"
	"github.com/sing3demons/saram-kafka/mongo"
	"github.com/sing3demons/saram-kafka/repository"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	logger.LoadLogConfig(logger.LogConfig{
		AppLog: logger.AppLog{
			LogConsole: true,
			LogFile:    true,
		},
		Summary: logger.SummaryLogConfig{
			LogFile:    true,
			LogConsole: true,
		},
		Detail: logger.DetailLogConfig{
			LogFile:    true,
			LogConsole: false,
		},
	})
}

const (
	ServiceRegisterTopic = "service.register"
	mongoUri             = "mongodb://localhost:27017/verify-service"
	servers              = "localhost:29092"
	groupID              = "example-group"
)

func main() {
	zapLogger := zap.New(
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
				MessageKey: "msg",
			}),
			zapcore.AddSync(os.Stdout),
			zap.InfoLevel))

	db, err := mongo.InitMongo(mongoUri, "example")
	if err != nil {
		zapLogger.Fatal("Failed to connect to MongoDB", zap.Error(err))
		os.Exit(1)
	}
	collection := db.Collection("example")
	repo := repository.NewRepository[handler.Example](collection)
	ms := microservice.NewApplication(servers, groupID, zapLogger)
	ms.Log("Starting microservice")

	ms.Consume(ServiceRegisterTopic, handler.NewHandler(repo).HandlerRegister)

	ms.Start()
}
