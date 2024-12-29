package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/sing3demons/logger-kp/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	gomail "gopkg.in/mail.v2"
)

// IMicroservice is interface for centralized service management
type IApplication interface {
	Start() error
	Stop()
	Cleanup() error
	Log(message string, fields ...map[string]any)

	// Consumer Services
	Consume(topic string, h ServiceHandleFunc) error
	NewProducer() sarama.SyncProducer
}

// Microservice is the centralized service management
type Microservice struct {
	exitChannel chan bool
	brokers     []string
	groupID     string
	client      sarama.ConsumerGroup
	Logger      *zap.Logger
	producer    *sarama.SyncProducer
}

type application struct {
	exitChannel chan bool
	brokers     []string
	groupID     string
	client      sarama.ConsumerGroup
	Logger      *zap.Logger
	producer    *sarama.SyncProducer
}

// IContext is the context for service
type IContext interface {
	L() *zap.Logger
	Log(message string, fields ...map[string]any)
	Param(name string) string
	Response(responseCode int, responseData interface{})
	ReadInput() string
	CommonLog(scenario string) (logger.DetailLog, logger.SummaryLog)
	// SendMail(message Message) error
	SendMessage(topic string, message interface{}, opts ...OptionProducerMessage) error
}

// ServiceHandleFunc is the handler for each Microservice
type ServiceHandleFunc func(ctx IContext) error

// NewMicroservice is the constructor function of Microservice
func NewApplication(brokers, groupID string) IApplication {
	return &application{
		brokers: strings.Split(brokers, ","),
		groupID: groupID,
		Logger:  logger.NewLogger(),
	}
}

func (ms *application) NewProducer() sarama.SyncProducer {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	config.Version = sarama.V2_5_0_0 // Set to Kafka version used
	producer, err := sarama.NewSyncProducer(ms.brokers, config)
	if err != nil {
		ms.Log(fmt.Sprintf("Error creating producer: %v", err))
		return nil
	}

	ms.producer = &producer
	return producer
}

// getProducer returns the producer
func (ms *application) getProducer() sarama.SyncProducer {
	if ms.producer == nil {
		return ms.NewProducer()
	}
	return *ms.producer
}

// Consume registers a consumer for the service
func (ms *application) Consume(topic string, h ServiceHandleFunc) error {
	if ms.client == nil {
		config := sarama.NewConfig()
		config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
		config.Version = sarama.V2_5_0_0 // Set to Kafka version used
		client, err := sarama.NewConsumerGroup(ms.brokers, ms.groupID, config)
		if err != nil {
			ms.Log(fmt.Sprintf("Error creating consumer group: %v", err))
			return err
		}

		ms.client = client
	}

	ms.Log(fmt.Sprintf("Consumer started for topic: [%s]", topic))

	defer ms.client.Close()

	handler := &ConsumerGroupHandler{
		ms:    ms,
		h:     h,
		topic: topic,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			if err := ms.client.Consume(ctx, []string{topic}, handler); err != nil {
				ms.Log(fmt.Sprintf("Error during consume: %v", err))
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	// Handle graceful shutdown
	consumptionIsPaused := false
	sigusr1 := make(chan os.Signal, 1)
	signal.Notify(sigusr1, syscall.SIGUSR1)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigterm:
		fmt.Println("Received termination signal. Initiating shutdown...")
		cancel()
	case <-ctx.Done():
		fmt.Println("terminating: context cancelled")
	case <-sigusr1:
		toggleConsumptionFlow(ms.client, &consumptionIsPaused)
	}
	// Wait for the consumer to finish processing
	wg.Wait()
	return nil
}

func toggleConsumptionFlow(client sarama.ConsumerGroup, isPaused *bool) {
	if *isPaused {
		client.ResumeAll()
		fmt.Println("Resuming consumption")
	} else {
		client.PauseAll()
		fmt.Println("Pausing consumption")
	}

	*isPaused = !*isPaused
}

// Start starts the microservice
func (ms *application) Start() error {
	osQuit := make(chan os.Signal, 1)
	ms.exitChannel = make(chan bool, 1)
	signal.Notify(osQuit, syscall.SIGTERM, syscall.SIGINT)
	exit := false
	for {

		if exit {
			break
		}
		select {
		case <-osQuit:
			exit = true
		case <-ms.exitChannel:
			exit = true
		}
	}
	return nil
}

// Stop stops the microservice
func (ms *application) Stop() {
	if ms.exitChannel == nil {
		return
	}
	ms.exitChannel <- true
}

// Cleanup performs cleanup before exit
func (ms *application) Cleanup() error {
	return nil
}

// Log logs a message to the console
func (ms *application) Log(message string, fields ...map[string]any) {
	var f []zapcore.Field
	for _, v := range fields {
		for key, value := range v {
			f = append(f, zap.Any(key, value))
		}
	}
	ms.Logger.Info(message, f...)
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

const (
	Key      = "logger"
	XSession = "X-Session-Id"
)

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

type RecordMetadata struct {
	TopicName      string `json:"topicName"`
	Partition      int32  `json:"partition"`
	ErrorCode      int    `json:"errorCode"`
	Offset         int64  `json:"offset,omitempty"`
	Timestamp      string `json:"timestamp,omitempty"`
	BaseOffset     string `json:"baseOffset,omitempty"`
	LogAppendTime  string `json:"logAppendTime,omitempty"`
	LogStartOffset string `json:"logStartOffset,omitempty"`
}

type OptionProducerMessage struct {
	key       string
	headers   []map[string]string
	Timestamp time.Time
	Metadata  interface{}
	Offset    int64
	Partition int32
}

type Payload struct {
	Header         any       `json:"header,omitempty"`
	Body           any       `json:"body"`
	Timestamp      time.Time // only set if kafka is version 0.10+, inner message timestamp
	BlockTimestamp time.Time // only set if kafka is version 0.10+, outer (compressed) block timestamp

	Topic     string
	Partition int32
	Offset    int64
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
