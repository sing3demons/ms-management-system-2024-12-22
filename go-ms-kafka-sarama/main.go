package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sing3demons/logger-kp/logger"
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

type Example struct {
	ID       string     `json:"id" bson:"_id"`
	Name     string     `json:"name" bson:"name"`
	CreateAt *time.Time `json:"create_at" bson:"create_at"`
	UpdateAt *time.Time `json:"update_at" bson:"update_at"`
	DeleteAt *time.Time `json:"-" bson:"delete_at,omitempty"`
}

type Register struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username"`
}
type InComingMessage struct {
	Body struct {
		Body Register `json:"body"`
	} `json:"body"`
}

func main() {

	db := mongo.InitMongo(mongoUri, "example")
	repo := repository.NewRepository[Example](db.Collection("example"))

	ms := microservice.NewApplication(servers, groupID)
	ms.Log("Starting microservice")
	err := ms.Consume(ServiceRegisterTopic, func(ctx microservice.IContext) error {
		c := context.Background()
		detailLog, summaryLog := ctx.CommonLog(ServiceRegisterTopic)

		summaryLog.AddSuccess("kafka_consumer", "register", "", "success")

		input := InComingMessage{}
		json.Unmarshal([]byte(ctx.ReadInput()), &input.Body)

		now := time.Now()
		doc := Example{
			ID:       uuid.New().String(),
			Name:     input.Body.Body.Username,
			CreateAt: &now,
			UpdateAt: &now,
			DeleteAt: nil,
		}

		repo.Create(c, &doc, detailLog, summaryLog)

		repo.FindOne(c, map[string]any{
			"_id": doc.ID,
		}, detailLog, summaryLog)

		upDateNow := time.Now()

		repo.UpdateOne(c, repository.Document[Example]{
			Filter: map[string]any{
				"_id": doc.ID,
			},
			New: Example{
				Name:     input.Body.Body.Username,
				UpdateAt: &upDateNow,
			},
			Options: map[string]any{
				"upsert": true,
			},
		}, detailLog, summaryLog)

		repo.Find(c, repository.Document[Example]{
			Filter: map[string]any{
				"delete_at": nil,
			},
			SortItems: map[string]any{
				"create_at": -1,
			},
			Projection: map[string]any{
				"_id":  1,
				"name": 1,
			},
		}, detailLog, summaryLog)

		err := ctx.SendMessage("service.verify", map[string]any{
			"email": input.Body.Body.Email,
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
