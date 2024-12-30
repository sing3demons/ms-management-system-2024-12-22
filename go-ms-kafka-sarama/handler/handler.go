package handler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/sing3demons/saram-kafka/microservice"
	"github.com/sing3demons/saram-kafka/repository"
)

type Handler interface {
	HandlerRegister(ctx microservice.IContext) error
}

type handler struct {
	repo *repository.Repository[Example]
}

const (
	ServiceRegisterTopic = "service.register"
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

func NewHandler(repo *repository.Repository[Example]) Handler {
	return &handler{repo: repo}
}

func (h *handler) HandlerRegister(ctx microservice.IContext) error {
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

	h.repo.Create(c, &doc, detailLog, summaryLog)

	h.repo.FindOne(c, map[string]any{
		"_id": doc.ID,
	}, detailLog, summaryLog)

	upDateNow := time.Now()

	h.repo.UpdateOne(c, repository.Document[Example]{
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

	h.repo.Find(c, repository.Document[Example]{
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
}
