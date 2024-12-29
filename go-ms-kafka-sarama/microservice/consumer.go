package microservice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/sing3demons/logger-kp/logger"
	"go.uber.org/zap"
	gomail "gopkg.in/mail.v2"
)

// ConsumerContext implements IContext
type ConsumerContext struct {
	ms       *application
	message  string
	msg      *sarama.ConsumerMessage
	l        logger.DetailLog
	s        logger.SummaryLog
	scenario *string
	invoke   *string
}

// NewConsumerContext is the constructor function for ConsumerContext
func NewConsumerContext(ms *application, msg *sarama.ConsumerMessage) *ConsumerContext {
	return &ConsumerContext{
		ms:      ms,
		msg:     msg,
		message: string(msg.Value),
	}
}

// Log logs a message
func (ctx *ConsumerContext) Log(message string, fields ...map[string]any) {
	ctx.ms.Log(message, fields...)
}

func (ctx *ConsumerContext) CommonLog(scenario string) (logger.DetailLog, logger.SummaryLog) {
	session := uuid.New().String()
	initInvoke := fmt.Sprintf("x-tid:%s", ctx.InitInvoke())

	detailLog := logger.NewDetailLog(session, initInvoke, scenario)
	summaryLog := logger.NewSummaryLog(session, initInvoke, scenario)

	// ctx.message convert to json
	data := Payload{
		Timestamp:      ctx.msg.Timestamp,
		Topic:          ctx.msg.Topic,
		Partition:      ctx.msg.Partition,
		Offset:         ctx.msg.Offset,
		BlockTimestamp: ctx.msg.BlockTimestamp,
	}
	json.Unmarshal(ctx.msg.Value, &data.Body)

	header := ctx.getHeaders()
	if header != nil {
		data.Header = header
	}

	detailLog.AddInputRequest("kafka_consumer", scenario, initInvoke, data, data)

	ctx.l = detailLog
	ctx.s = summaryLog
	ctx.invoke = &initInvoke
	ctx.scenario = &scenario

	return ctx.l, ctx.s
}

func (ctx *ConsumerContext) L() *zap.Logger {
	fmt.Println("Logger")
	c := context.Background()
	// logger.InitSession(c, ctx.ms.Logger)
	pCtx, _log := logger.InitSession(c, ctx.ms.Logger)
	c = pCtx

	return _log
}

func (ctx *ConsumerContext) SendMessage(topic string, message interface{}, opts ...OptionProducerMessage) error {
	timestamp := time.Now()
	producer := ctx.ms.getProducer()
	if producer == nil {
		producer = ctx.ms.NewProducer()
		if producer == nil {
			return fmt.Errorf("error creating producer")
		}
	}

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic:     topic,
		Value:     sarama.StringEncoder(data),
		Timestamp: timestamp,
	}

	if len(opts) > 0 {
		for _, opt := range opts {
			if opt.key != "" {
				msg.Key = sarama.StringEncoder(opt.key)
			}

			if len(opt.headers) > 0 {
				for _, header := range opt.headers {
					for key, value := range header {
						msg.Headers = append(msg.Headers, sarama.RecordHeader{
							Key:   []byte(key),
							Value: []byte(value),
						})
					}
				}
			}

			if !opt.Timestamp.IsZero() {
				msg.Timestamp = opt.Timestamp
			}

			if opt.Metadata != nil {
				msg.Metadata = opt.Metadata
			}

			if opt.Offset > 0 {
				msg.Offset = opt.Offset
			}

			if opt.Partition > 0 {
				msg.Partition = opt.Partition
			}
		}
	}

	invoke := logger.GenerateXTid(topic)

	if ctx.l != nil {
		ctx.l.AddOutputRequest("kafka_producer", topic, invoke, msg, msg)
		ctx.l.End()
	}

	partition, offset, err := producer.SendMessage(msg)
	if err != nil {
		ctx.l.AddOutputRequest("kafka_producer", topic, invoke, err.Error(), map[string]any{"error": err.Error()})
		ctx.s.AddError("kafka_producer", topic, invoke, err.Error())
		return err
	}
	recordMetadata := RecordMetadata{
		TopicName:      topic,
		Partition:      partition,
		Offset:         offset,
		ErrorCode:      0,
		Timestamp:      timestamp.String(),
		BaseOffset:     "",
		LogAppendTime:  "",
		LogStartOffset: "",
	}

	ctx.l.AddInputRequest("kafka_producer", topic, invoke, "", recordMetadata)

	return nil
}

// Param returns a parameter by name (not used in this example)
func (ctx *ConsumerContext) Param(name string) string {
	return ""
}

// ReadInput returns the message
func (ctx *ConsumerContext) ReadInput() string {
	return ctx.message
}

func (ctx *ConsumerContext) getHeaders() map[string]string {

	headers := ctx.msg.Headers
	if headers != nil {
		headerMap := make(map[string]string)
		for _, header := range headers {
			headerMap[string(header.Key)] = string(header.Value)
		}
		return headerMap
	}
	return nil
}

// Response returns a response to the client (not used in this example)
func (ctx *ConsumerContext) Response(responseCode int, responseData interface{}) {
	if ctx.l != nil {
		log := ctx.l
		log.AddOutputResponse("kafka_consumer", *ctx.scenario, *ctx.invoke, responseData, responseData)
		log.AutoEnd()
		ctx.l = nil
	}

	if ctx.s != nil {
		s := *&ctx.s
		if !s.IsEnd() {
			resultCode := strconv.Itoa(responseCode)
			resultDesc := strings.ReplaceAll(http.StatusText(responseCode), " ", "_")
			s.End(resultCode, resultDesc)
			s = nil
		}
	}

	ctx.scenario = nil
	ctx.invoke = nil
	return
}

func (ctx *ConsumerContext) InitInvoke() string {
	v7, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return v7.String()
}

type MailServer struct {
	Host   string `json:"host,omitempty"`
	Port   int    `json:"port,omitempty"`
	Secure bool   `json:"secure,omitempty"`
	Auth   *Auth  `json:"auth,omitempty"`
}

type Auth struct {
	User string `json:"username,omitempty"`
	Pass string `json:"password,omitempty"`
}

type Message struct {
	From        string   `json:"from,omitempty"`
	To          string   `json:"to,omitempty"`
	Subject     string   `json:"subject,omitempty"`
	Body        string   `json:"body,omitempty"`
	Attachment  string   `json:"attachment,omitempty"`
	Attachments []string `json:"attachments,omitempty"`
}

type Result struct {
	Err        bool
	ResultDesc string
	ResultData interface{}
}

func (ctx *ConsumerContext) SendMail(mailServer MailServer, params Message) Result {
	cmdName := "send_mail"
	node := "mail_server"
	result := Result{}
	invoke := logger.GenerateXTid(cmdName)
	if strings.Contains(params.From, "{email_from}") {
		if mailServer.Auth != nil && mailServer.Auth.User != "" {
			params.From = strings.ReplaceAll(params.From, "{email_from}", mailServer.Auth.User)
		}
	}

	// Setup SMTP configuration
	dialer := gomail.NewDialer(mailServer.Host, mailServer.Port, "", "")
	if mailServer.Auth != nil && mailServer.Auth.User != "" && mailServer.Auth.Pass != "" {
		dialer.Username = mailServer.Auth.User
		dialer.Password = mailServer.Auth.Pass
	}
	dialer.SSL = mailServer.Secure

	ctx.l.AddOutputRequest(node, cmdName, invoke, params, params)
	ctx.l.End()

	// Create the email message
	message := gomail.NewMessage()
	message.SetHeader("From", params.From)
	message.SetHeader("To", params.To)
	message.SetHeader("Subject", params.Subject)
	message.SetBody("text/html", params.Body)
	message.SetHeader("Return-Path", params.From)

	if params.Attachment != "" {
		message.Attach(params.Attachment)
	}

	// Sending the email
	// log.Printf("smtpBody: Host=%s, Port=%d, Secure=%t, User=%s", mailServer.Host, mailServer.Port, mailServer.Secure, dialer.Username)

	err := dialer.DialAndSend(message)
	if err != nil {
		if strings.Contains(err.Error(), "timeout") {
			result.ResultDesc = "timeout"
		} else {
			result.ResultDesc = "connection_error"
		}
		result.Err = true
		result.ResultData = err.Error()

		ctx.l.AddInputRequest(node, cmdName, invoke, result, result)
		ctx.s.AddError(node, cmdName, "500", result.ResultDesc)
		return result
	}

	result.ResultData = "Email sent successfully"
	ctx.l.AddInputRequest(node, cmdName, invoke, result, result)
	ctx.s.AddSuccess(node, cmdName, "200", "success")

	return result
}

// ConsumerGroupHandler implements sarama.ConsumerGroupHandler
type ConsumerGroupHandler struct {
	ms    *application
	h     ServiceHandleFunc
	topic string
}

func (handler *ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}
func (handler *ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim processes messages from a claim
func (handler *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		// set context
		xTid := fmt.Sprintf("x-tid:%s:s%s", uuid.New().String(), strconv.Itoa(int(session.GenerationID())))
		l := handler.ms.Logger.With(zap.String(XSession, xTid))
		handler.ms.Logger = l

		handler.ms.Log(fmt.Sprintf("Consumer: %s", msg.Topic))
		ctx := NewConsumerContext(handler.ms, msg)
		if err := handler.h(ctx); err != nil {
			handler.ms.Log(fmt.Sprintf("Consumer error: %v", err))
		}
		session.MarkMessage(msg, "")
	}
	return nil
}
