package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/sing3demons/profile-service/constants"
	"github.com/sing3demons/profile-service/ms"
	"github.com/sing3demons/profile-service/store"
	"github.com/sing3demons/profile-service/template"
	"github.com/sing3demons/profile-service/utils"
)

type Profile struct {
	ID           string `json:"id"`
	Href         string `json:"href,omitempty"`
	Email        string `json:"email,omitempty"`
	UserName     string `json:"username,omitempty"`
	Password     string `json:"password,omitempty"`
	FirstName    string `json:"first_name,omitempty"`
	LastName     string `json:"last_name,omitempty"`
	DateOfBirth  string `json:"date_of_birth,omitempty"`
	PhoneNumber  string `json:"phone_number,omitempty"`
	Gender       string `json:"gender,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
	CreatedBy    string `json:"created_by,omitempty"`
	UpdatedBy    string `json:"updated_by,omitempty"`
	DisplayName  string `json:"display_name,omitempty"`
	ProfileImage string `json:"profile_image,omitempty"`
}

func init() {
	// load env
	godotenv.Load(".env")

	if os.Getenv("SERVICE_NAME") == "" {
		projectName := utils.ProjectName()
		os.Setenv("SERVICE_NAME", projectName)
	}
}

func main() {
	app := ms.NewApplication(ms.Config{
		Addr: "8080",
		Env:  "local",
		Name: utils.ProjectName(),
		KafkaCfg: ms.KafkaConfig{
			Brokers: "localhost:29092",
			GroupID: "profile-service",
		},
		Db: ms.DbConfig{
			Addr:         os.Getenv("DATABASE_URL"),
			MaxOpenConns: 10,
			MaxIdleConns: 10,
			MaxIdleTime:  "5s",
			Driver:       "postgres",
		},
		LogConfig: ms.LogConfig{
			ProjectName: "profile-service",
			AppLog: ms.AppLog{
				LogFile: true,
			},
			Detail: ms.DetailLogConfig{
				RawData:    true,
				LogFile:    true,
				LogConsole: false,
			},
			Summary: ms.SummaryLogConfig{
				RawData:    true,
				LogFile:    true,
				LogConsole: true,
			},
		},
		MailServer: ms.MailServer{
			Host:   "localhost",
			Port:   1025,
			Secure: false,
		},
	})

	app.POST("/mail", func(ctx ms.IContext) error {
		fmt.Println("send mail")
		payload := ctx.ReadInput()

		cmd := "send_mail"
		node := "consume"
		initInvoke := ms.GenerateXTid("profile")
		scenario := "send_mail"

		_, summaryLog := ctx.CommonLog(initInvoke, scenario, "anonymous")
		summaryLog.AddSuccessBlock(node, cmd, "200", "success")

		var body ms.Message
		if err := mapToStruct(payload.Body, &body); err != nil {
			summaryLog.AddErrorBlock(node, cmd, "400", "invalid_request")
			summaryLog.AddField("error", err.Error())
			return ctx.Response(400, err.Error())

		}

		ctx.SendMail(ms.Message{
			From:    body.From,
			To:      body.To,
			Subject: body.Subject,
			Body:    template.ConfirmEmailMessage(body.Body),
		})

		return ctx.Response(200, "success")
	})

	producer := app.NewProducer()

	// Create table
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS Profile (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE,
			username VARCHAR(255) UNIQUE,
			password VARCHAR(255),
			first_name VARCHAR(100),
			last_name VARCHAR(100),
			date_of_birth DATE,
			phone_number VARCHAR(15),
			gender VARCHAR(50),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_by VARCHAR(100),
			updated_by VARCHAR(100),
			display_name VARCHAR(255),
			profile_image TEXT
		);
		`

	conn := app.ConnDatabase(createTableQuery)

	s := store.NewStorer(conn)

	// handler
	h := Handler{s}

	// get by id
	app.GET("/users/{publicId}", h.GetUserByPublicId)

	app.POST("/health", func(ctx ms.IContext) error {
		return ctx.Response(200, "OK")
	})

	app.Consume("service.auth", func(cc ms.IContext) error {
		fmt.Println("consume xxx", cc)
		return nil
	})

	app.Consume("service.register", func(ctx ms.IContext) error {
		payload := ctx.ReadInput()

		cmd := "register"
		node := "consume"
		initInvoke := ms.GenerateXTid("profile")
		scenario := "service.register"

		detailLog, summaryLog := ctx.CommonLog(initInvoke, scenario, "anonymous")

		summaryLog.AddSuccessBlock(node, cmd, "200", "success")

		var body Register
		if err := mapToStruct(payload.Body, &body); err != nil {
			summaryLog.AddErrorBlock(node, cmd, "400", "invalid_request")
			summaryLog.AddField("error", err.Error())
			return ctx.Response(400, err.Error())

		}

		c := context.Background()
		tx, _ := conn.BeginTx(c, nil)
		p := store.User{}
		p.Password.Set(body.Password)
		user := &store.User{
			Username: body.Username,
			Email:    body.Email,
			Password: p.Password,
		}
		s.Users.Create(c, tx, user, detailLog, summaryLog)

		tx.Commit()

		producer.SendMessage("service.verify", "", user, detailLog, summaryLog)

		// ctx.SendMail(ms.Message{
		// 	From:    "dev@test.com",
		// 	To:      body.Email,
		// 	Subject: "Register Success",
		// 	Body:    template.ConfirmEmailMessage("http://localhost:8080/confirm"),
		// })

		return ctx.Response(200, "success")

	})

	app.Run()
	defer app.CleanUp()
}

func mapToStruct[T any](data interface{}, result T) error {
	// Assert the data is a map[string]interface{}
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshaling data:", err)
		return err
	}

	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		fmt.Println("Error unmarshaling data:", err)
		return err
	}
	return nil
}

type Register struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username"`
}

type InComing[T any] struct {
	Header struct {
		Session string `json:"session"`
	} `json:"header"`
	Body T `json:"body"`
}

type Handler struct {
	*store.Storer
}

func (s Handler) GetUserByPublicId(c ms.IContext) error {
	initInvoke := ms.GenerateXTid("profile")
	cmd := "get_user_by_id"

	detailLog, summaryLog := c.CommonLog(initInvoke, cmd, "anonymous")
	summaryLog.AddSuccessBlock(constants.CLIENT, cmd, "200", "success")

	var getUser store.User

	publicId := c.Param("publicId")
	if utils.IsEmail(publicId) {
		user, err := s.Users.GetByEmail(context.Background(), publicId, detailLog, summaryLog)
		if err != nil {
			return c.Response(500, err)
		}
		getUser = *user
	} else {
		user, err := s.Users.GetByID(context.Background(), publicId, detailLog, summaryLog)
		if err != nil {
			detailLog.AddOutputResponse(constants.CLIENT, cmd, initInvoke, nil, err)
			return c.Response(500, err)
		}
		getUser = *user
	}

	// var optionAttributes []http_service.RequestAttributes
	// for i := 1; i <= 1; i++ {
	// 	optionAttribute := http_service.RequestAttributes{
	// 		Command: "get_x",
	// 		Service: "node",
	// 		Invoke:  "get_all_users",
	// 		Method:  http.MethodGet,
	// 		URL:     "http://localhost:3000/x/{id}",
	// 		Headers: map[string]string{
	// 			"Content-Type": "application/json",
	// 		},
	// 		Params: map[string]string{
	// 			"id": fmt.Sprintf("%d", i),
	// 		},
	// 		Timeout: 10,
	// 	}
	// 	optionAttributes = append(optionAttributes, optionAttribute)
	// }

	// 	_, err := http_service.RequestHttp(optionAttributes, detailLog, summaryLog)

	resp := Profile{
		ID:           getUser.ID,
		Href:         "http://localhost:8080/users/" + getUser.ID,
		Email:        getUser.Email,
		UserName:     getUser.Username,
		FirstName:    getUser.FirstName,
		LastName:     getUser.LastName,
		DateOfBirth:  getUser.DateOfBirth,
		PhoneNumber:  getUser.PhoneNumber,
		Gender:       getUser.Gender,
		DisplayName:  getUser.DisplayName,
		ProfileImage: getUser.ProfileImage,
	}

	return c.Response(200, resp)
}
