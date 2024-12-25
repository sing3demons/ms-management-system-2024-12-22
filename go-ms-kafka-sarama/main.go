package main

import (
	"fmt"

	"github.com/sing3demons/logger-kp/logger"
)

func init() {
	logger.LoadLogConfig(logger.LogConfig{
		Summary: logger.SummaryLogConfig{
			LogFile:    false,
			LogConsole: true,
		},
		// Detail: logger.DetailLogConfig{
		// 	LogFile:    false,
		// 	LogConsole: false,
		// },
	})
}

const (
	ServiceRegisterTopic = "service.register"
)

func main() {
	servers := "localhost:29092"
	groupID := "example-group"

	ms := NewApplication(servers, groupID)
	ms.Log("Starting microservice")
	err := ms.Consume(ServiceRegisterTopic, func(ctx IContext) error {
		appLog := ctx.L()
		appLog.Info("Registering service")
		_, summaryLog := ctx.CommonLog(ServiceRegisterTopic)

		summaryLog.AddSuccess("kafka_consumer", "register", "", "success")

		err := ctx.SendMessage("service.verify", map[string]any{
			"email": "test@dev.com",
		})
		if err != nil {
			ctx.Response(500, map[string]any{
				"error": err.Error(),
			})
			return err
		}

		summaryLog.AddSuccess("kafka_producer", "service.verify", "", "success")

		ctx.Response(200, map[string]any{
			"message": "success",
		})
		return nil
	})
	if err != nil {
		fmt.Println("Error:", err)
	}

	ms.Start()
}
