package ms

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/sing3demons/profile-service/constants"
	"github.com/sing3demons/profile-service/logger"
	"github.com/sing3demons/profile-service/utils"
)

// NewConsumerContext is the constructor function for ConsumerContext
func NewConsumerContext(message kafkaMessage, ms *application) IContext {
	return &ConsumerContext{
		message: message,
		ms:      ms,
	}
}

func (h *ConsumerContext) CommonLog(initInvoke, cmd, identity string) (logger.DetailLog, logger.SummaryLog) {
	if utils.IsStructEmpty(h.payload) {
		h.payload = h.Payload()
	}
	if h.payload.Header.Session == "" {
		h.payload.Header.Session = fmt.Sprintf("%s-%s", cmd, uuid.New().String())
	}
	req := &http.Request{}
	req = req.WithContext(context.WithValue(req.Context(), constants.Session, h.payload.Header.Session))

	conf := logger.LogConfig{}
	conf.ProjectName = h.ms.config.LogConfig.ProjectName
	conf.Namespace = h.ms.config.LogConfig.Namespace
	conf.Summary.RawData = h.ms.config.LogConfig.Summary.RawData
	conf.Summary.LogFile = h.ms.config.LogConfig.Summary.LogFile
	conf.Summary.LogConsole = h.ms.config.LogConfig.Summary.LogConsole
	conf.Summary.LogSummary = h.ms.config.LogConfig.Summary.LogSummary

	conf.Detail.RawData = h.ms.config.LogConfig.Detail.RawData
	conf.Detail.LogFile = h.ms.config.LogConfig.Detail.LogFile
	conf.Detail.LogConsole = h.ms.config.LogConfig.Detail.LogConsole
	conf.Detail.LogDetail = h.ms.config.LogConfig.Detail.LogDetail

	detailLog := logger.NewDetailLog(req, initInvoke, cmd, identity, conf)
	summaryLog := logger.NewSummaryLog(req, initInvoke, cmd, conf)

	detailLog.AddInputRequest(constants.CLIENT, cmd, initInvoke, nil, h.payload)
	h.l = detailLog
	h.s = summaryLog
	h.invoke = initInvoke
	h.topic = cmd
	return h.l, h.s
}

func (h *ConsumerContext) SendKafkaMessage(topic string, payload any) error {

	return nil
}

// Log will log a message
func (ctx *ConsumerContext) Log(message string) {
	fmt.Println("Consumer: ", message)
}

// Param return parameter by name (empty in case of Consumer)
func (ctx *ConsumerContext) Param(name string) string {
	return ""
}

func (ctx *ConsumerContext) ReadInput() InComing {
	data := InComing{}

	payload := Payload{}
	if err := json.Unmarshal([]byte(ctx.message.value), &payload); err != nil {
		msg := Msg{
			Topic: ctx.topic,
			Header: Header{
				Session: GenerateXTid(ctx.topic),
			},
			Body: "",
		}
		data.Headers = map[string]interface{}{
			"session": msg.Header.Session,
		}
		data.Body = msg.Body
		ctx.payload = msg
		return data
	}

	if payload.Header.Session == "" {
		payload.Header.Session = GenerateXTid(ctx.topic)
	}

	msg := Msg{
		Topic:  ctx.topic,
		Header: payload.Header,
		Body:   payload.Body,
	}
	data.Headers = map[string]interface{}{
		"session": msg.Header.Session,
	}
	data.Body = msg.Body
	ctx.payload = msg
	return data
}

func (ctx *ConsumerContext) Payload() Msg {
	ctx.topic = ctx.message.topic
	fmt.Println(fmt.Sprintf("Consumer [%v] -> Payload: [%s]\n", ctx.topic, ctx.message.value))

	payload := Payload{}
	if err := json.Unmarshal([]byte(ctx.message.value), &payload); err != nil {
		data := Msg{
			Topic: ctx.topic,
			Header: Header{
				Session: GenerateXTid(ctx.topic),
			},
			Body: "",
		}
		ctx.payload = data
		return data
	}

	if payload.Header.Session == "" {
		payload.Header.Session = GenerateXTid(ctx.topic)
	}

	data := Msg{
		Topic:  ctx.topic,
		Header: payload.Header,
		Body:   payload.Body,
	}
	ctx.payload = data

	return data
}

// Response return response to client
func (ctx *ConsumerContext) Response(responseCode int, responseData interface{}) error {

	if ctx.l != nil {
		log := ctx.l
		log.AddOutputResponse("consume", ctx.topic, ctx.invoke, responseData, responseData)
		log.AutoEnd()
		ctx.l = nil
	}

	if ctx.s != nil {
		s := *&ctx.s
		if !s.IsEnd() {
			resultCode := fmt.Sprintf("%d", responseCode)
			resultDesc := http.StatusText(responseCode)
			s.End(resultCode, resultDesc)
			s = nil
		}
	}

	log.Println(fmt.Sprintf("Consumer [%v] -> Response: [%s]\n", ctx.topic, responseData))
	ctx = nil

	return nil
}
func (h *ConsumerContext) SendMail(message Message) error {
	result := sendMail(h.ms.config.MailServer, message, h.l, h.s)
	if result.Err {
		return fmt.Errorf("Error sending email: %s", result.ResultDesc)
	}
	return nil
}
