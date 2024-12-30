package microservice

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/sing3demons/logger-kp/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

const (
	Key      = "logger"
	XSession = "X-Session-Id"
)

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
func NewApplication(brokers, groupID string, log ...*zap.Logger) IApplication {
	app := &application{
		brokers: strings.Split(brokers, ","),
		groupID: groupID,
		Logger:  logger.NewLogger(),
	}

	if len(log) > 0 {
		app.Logger = log[0]
	}

	return app
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
