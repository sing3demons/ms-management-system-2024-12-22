package ms

import (
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func (ms *application) newKafkaConsumer(servers string, groupID string) (*kafka.Consumer, error) {
	// Configurations
	// https://github.com/edenhill/librdkafka/blob/master/CONFIGURATION.md
	config := &kafka.ConfigMap{

		// Alias for metadata.broker.list: Initial list of brokers as a CSV list of broker host or host:port.
		// The application may also use rd_kafka_brokers_add() to add brokers during runtime.
		"bootstrap.servers": servers,

		// Client group id string. All clients sharing the same group.id belong to the same group.
		"group.id": groupID,

		// Action to take when there is no initial offset in offset store or the desired offset is out of range:
		// 'smallest','earliest' - automatically reset the offset to the smallest offset,
		// 'largest','latest' - automatically reset the offset to the largest offset,
		// 'error' - trigger an error which is retrieved by consuming messages and checking 'message->err'.
		"auto.offset.reset": "earliest",

		// Protocol used to communicate with brokers.
		// plaintext, ssl, sasl_plaintext, sasl_ssl
		"security.protocol": "plaintext",

		// Automatically and periodically commit offsets in the background.
		// Note: setting this to false does not prevent the consumer from fetching previously committed start offsets.
		// To circumvent this behaviour set specific start offsets per partition in the call to assign().
		"enable.auto.commit": true,

		// The frequency in milliseconds that the consumer offsets are committed (written) to offset storage. (0 = disable).
		// default = 5000ms (5s)
		// 5s is too large, it might cause double process message easily, so we reduce this to 200ms (if we turn on enable.auto.commit)
		"auto.commit.interval.ms": 500,

		// Automatically store offset of last message provided to application.
		// The offset store is an in-memory store of the next offset to (auto-)commit for each partition
		// and cs.Commit() <- offset-less commit
		"enable.auto.offset.store": true,

		// Enable TCP keep-alives (SO_KEEPALIVE) on broker sockets
		"socket.keepalive.enable": true,
	}

	kc, err := kafka.NewConsumer(config)
	if err != nil {
		return nil, err
	}
	return kc, err
}

type consumerContext struct {
	topic       string
	topics      []string
	readTimeout time.Duration
}

type kafkaMessage struct {
	topic     string
	timestamp time.Time
	key       string
	value     string
}

func (ms *application) consumeSingle(ctx consumerContext, h ServiceHandleFunc) {
	c, err := ms.newKafkaConsumer(ms.config.KafkaCfg.Brokers, ms.config.KafkaCfg.GroupID)
	if err != nil {
		ms.Log("Consumer", err.Error())
		return
	}
	defer c.Close()

	c.Subscribe(ctx.topic, nil)

	for {
		ms.processMessage(ctx, c, h)
	}
}
func (ms *application) consumeMultiple(ctx consumerContext, h ServiceHandleFunc) {
	c, err := ms.newKafkaConsumer(ms.config.KafkaCfg.Brokers, ms.config.KafkaCfg.GroupID)
	if err != nil {
		ms.Log("Consumer", err.Error())
		return
	}
	c.SubscribeTopics(ctx.topics, nil)

	for {
		ms.processMessage(ctx, c, h)
	}
}

func (ms *application) processMessage(ctx consumerContext, c *kafka.Consumer, h ServiceHandleFunc) {
	if ctx.readTimeout <= 0 {
		// readtimeout -1 indicates no timeout
		ctx.readTimeout = -1
	}

	msg, err := c.ReadMessage(ctx.readTimeout)
	if err != nil {
		ms.handleKafkaError(ctx, err)
		return
	}

	// Execute Handler
	h(NewConsumerContext(kafkaMessage{
		topic:     *msg.TopicPartition.Topic,
		timestamp: msg.Timestamp,
		value:     string(msg.Value),
		key:       string(msg.Key),
	}, ms))
}

func (ms *application) handleKafkaError(ctx consumerContext, err error) {
	kafkaErr, ok := err.(kafka.Error)
	if ok {
		if kafkaErr.Code() == kafka.ErrTimedOut {
			if ctx.readTimeout == -1 {
				// No timeout just continue to read message again
				return
			}
		}
	}
	ms.Log("Consumer", err.Error())
}

// Consume register service endpoint for Consumer service
func (ms *application) Consume(topic string, h ServiceHandleFunc) error {
	// if ms.consumer == nil {
	// 	ms.Log("Consumer", fmt.Sprintf("Consumer is not initialized for topic %s", topic))
	// 	return errors.New("consumer is not initialized")
	// }
	go ms.consumeSingle(consumerContext{
		topic:       topic,
		readTimeout: time.Duration(-1),
	}, h)
	return nil
}
