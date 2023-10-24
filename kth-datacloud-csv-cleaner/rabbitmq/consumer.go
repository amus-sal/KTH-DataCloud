package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	queueName            = "datacloud-csv-split"
	deadLetterExchange   = "datacloud.dlx"
	webhookReceivedEvent = "csv.partition.created"
	requeueDelay         = 60000 // 1 minute.
	requeueLimit         = 3
	prefetchCount        = 1
)

type connector interface {
	NotifyConnection() <-chan *amqp.Connection
	Reconnect()
}

type eventService interface {
	Handle(ctx context.Context, eventID string, filePath string) error
}

// Consumer represents a RabbitMQ consumer.
type Consumer struct {
	exchange     string
	connector    connector
	ch           *amqp.Channel
	eventService eventService
	logger       logger
}

// NewConsumer returns a new consumer with all dependencies set up.
func NewConsumer(
	connector connector,
	exchange string,
	eventService eventService,
	logger logger,
) Consumer {
	return Consumer{
		exchange:     exchange,
		connector:    connector,
		eventService: eventService,
		logger:       logger,
	}
}

// Start will receive a channel with an open amqp connection
// and handling the topology setup.
func (c *Consumer) Start() error {
	for conn := range c.connector.NotifyConnection() {
		// Got a new connection, we have to setup our topology.
		if err := c.setup(conn); err != nil {
			// Force a reconnect to request a new connection.
			c.connector.Reconnect()

			// Log error.
			fmt.Println(err)

			continue
		}
	}

	return errors.New("consumer connection notifier exited")
}

func (c *Consumer) setup(conn *amqp.Connection) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}

	// Set the healthy channel on the consumer.
	c.ch = ch

	err = c.ch.Qos(
		prefetchCount, // prefetch count: how many messages will be delivered at once before ack is received.
		0,             // prefetch size: if greater than 0 will try to keep at least that many bytes of deliveries flushed to the network.
		false,         // global: if this should apply to all existing consumers on all channels on the same connection.
	)
	if err != nil {
		return err
	}

	// Before anything else, let's declare a dead letter exchange
	// where nacked messages that aren't requeued can be stored.
	if err = c.declareDeadLetterExchange(ch); err != nil {
		return err
	}

	// Before we bind the queue to an exchange, make sure it exists
	// and is of the correct type and configuration.
	err = ch.ExchangeDeclare(
		c.exchange,          // name
		"x-delayed-message", // type
		true,                // durable
		false,               // auto-deleted
		false,               // internal
		false,               // no-wait
		map[string]interface{}{
			"x-delayed-type": "topic",
		},
	)
	if err != nil {
		return err
	}

	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		map[string]interface{}{
			"x-queue-type":           "quorum",
			"x-dead-letter-exchange": deadLetterExchange,
		},
	)
	if err != nil {
		return err
	}

	err = ch.QueueBind(
		q.Name,               // queue name
		webhookReceivedEvent, // routing key
		c.exchange,           // exchange
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Start listening on the messages from this new channel.
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return err
	}

	// Start listening for incoming messages on the channel.
	go listen(msgs, c)

	return nil
}

// declareDeadLetterExchange will setup an exchange and a queue to receive all dead messages on.
func (c *Consumer) declareDeadLetterExchange(ch *amqp.Channel) error {
	// The type fanout means any message regardless of routing key can be stored on the exchange.
	if err := ch.ExchangeDeclare(
		deadLetterExchange,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	return nil
}

type FileEvent struct {
	EventID  string `json:"event_id"`
	FilePath string `json:"file_path"`
}

func (c *Consumer) csvConverter(msg *amqp.Delivery) {

	var payload FileEvent
	if err := json.Unmarshal(msg.Body, &payload); err != nil {

		// If something is wrong in the json payload, just send the message to dlx.
		if err := msg.Nack(false, false); err != nil {
			fmt.Println(err)
		}

		return
	}

	if err := c.eventService.Handle(context.Background(), payload.EventID, payload.FilePath); err != nil {
		fmt.Println(err)
		return
	}

	if err := msg.Ack(false); err != nil {
		fmt.Println(err)
	}
}

// listen will stall indefinitely and listen for incoming messages,
// when the connection dies the scope will close, only for the
// consumer to open a new one.
func listen(messages <-chan amqp.Delivery, c *Consumer) {
	for msg := range messages {
		// Create a new memory address to solve the loop issue.
		clone := msg
		switch msg.RoutingKey {
		case webhookReceivedEvent:
			go c.csvConverter(&clone)
		}

	}
}
