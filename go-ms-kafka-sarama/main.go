package main

import (
	"github.com/sing3demons/logger-kp/logger"
	"github.com/sing3demons/saram-kafka/handler"
	"github.com/sing3demons/saram-kafka/microservice"
	"github.com/sing3demons/saram-kafka/mongo"
	"github.com/sing3demons/saram-kafka/repository"
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

	db := mongo.InitMongo(mongoUri, "example")
	repo := repository.NewRepository[handler.Example](db.Collection("example"))
	h := handler.NewHandler(repo)

	ms := microservice.NewApplication(servers, groupID)
	ms.Log("Starting microservice")

	ms.Consume(ServiceRegisterTopic, h.HandlerRegister)

	ms.Start()
}
