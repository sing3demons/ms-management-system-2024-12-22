package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/sing3demons/saram-kafka/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// IMicroservice is interface for centralized service management
type IMicroservice interface {
	Start() error
	Stop()
	Cleanup() error
	Log(message string, fields ...map[string]any)

	// Consumer Services
	Consume(servers string, topic string, groupID string, h ServiceHandleFunc) error
}

// Microservice is the centralized service management
type Microservice struct {
	exitChannel chan bool
	brokers     []string
	groupID     string
	client      sarama.ConsumerGroup
	Logger      *zap.Logger
}

// IContext is the context for service
type IContext interface {
	Log(message string, fields ...map[string]any)
	Param(name string) string
	Response(responseCode int, responseData interface{})
	ReadInput() string
}

// ServiceHandleFunc is the handler for each Microservice
type ServiceHandleFunc func(ctx IContext) error

// NewMicroservice is the constructor function of Microservice
func NewMicroservice(brokers, groupID string) *Microservice {
	return &Microservice{
		brokers: strings.Split(brokers, ","),
		groupID: groupID,
		Logger:  logger.NewLog(false, true),
	}
}

// Consume registers a consumer for the service
func (ms *Microservice) Consume(topic string, h ServiceHandleFunc) error {
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
func (ms *Microservice) Start() error {
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
func (ms *Microservice) Stop() {
	if ms.exitChannel == nil {
		return
	}
	ms.exitChannel <- true
}

// Cleanup performs cleanup before exit
func (ms *Microservice) Cleanup() error {
	return nil
}

// Log logs a message to the console
func (ms *Microservice) Log(message string, fields ...map[string]any) {
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
	ms    *Microservice
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

		handler.ms.Log(fmt.Sprintf("Consumer: %s", string(msg.Value)))
		ctx := NewConsumerContext(handler.ms, string(msg.Value))
		if err := handler.h(ctx); err != nil {
			handler.ms.Log(fmt.Sprintf("Consumer error: %v", err))
		}
		session.MarkMessage(msg, "")
	}
	return nil
}

// ConsumerContext implements IContext
type ConsumerContext struct {
	ms      *Microservice
	message string
}

// NewConsumerContext is the constructor function for ConsumerContext
func NewConsumerContext(ms *Microservice, message string) *ConsumerContext {
	return &ConsumerContext{
		ms:      ms,
		message: message,
	}
}

// Log logs a message
func (ctx *ConsumerContext) Log(message string, fields ...map[string]any) {
	ctx.ms.Log(message, fields...)
}

// Param returns a parameter by name (not used in this example)
func (ctx *ConsumerContext) Param(name string) string {
	return ""
}

// ReadInput returns the message
func (ctx *ConsumerContext) ReadInput() string {
	return ctx.message
}

// Response returns a response to the client (not used in this example)
func (ctx *ConsumerContext) Response(responseCode int, responseData interface{}) {
	return
}
