package microservice

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/IBM/sarama"
)

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
