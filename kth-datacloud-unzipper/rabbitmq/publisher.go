package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/amus-sal/kth-datacloud-unzipper/domain"
	amqp "github.com/rabbitmq/amqp091-go"
)

type logger interface {
	Errorw(msg string, keysAndValues ...interface{})
}

type connector interface {
	NotifyConnection() <-chan *amqp.Connection
	Reconnect()
}

// tracingFromContext func will extract tracing information from the context,
// this information can then be used to pass as headers over AMQP.
type tracingFromContext func(ctx context.Context) map[string]string

type Publisher struct {
	connector   connector
	exchange    string
	channelLock sync.Mutex
	channel     *amqp.Channel
	closer      chan struct{}
	once        sync.Once
}

func NewPublisher(connector connector, exchange string, logger logger) *Publisher {
	p := Publisher{
		connector: connector,
		exchange:  exchange,
		closer:    make(chan struct{}),
	}

	// Listen for changes to the connection.
	go func() {
		notifier := connector.NotifyConnection()
		for {
			select {
			case <-p.closer:
				return
			case connection, ok := <-notifier:
				// Signal from connection notifier.
				if !ok {
					// notifier closed its channel, we should also exit.
					p.Close()
					return
				}

				if connection != nil {
					// New connection, set the channel on the publisher.
					if channel, err := connection.Channel(); err == nil {
						p.channelLock.Lock()
						p.channel = channel
						p.channelLock.Unlock()
					}

					// Setup topology required to publish messages.
					err := p.channel.ExchangeDeclare(
						p.exchange,          // name
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
						// Force a reconnect to request a new connection.
						p.connector.Reconnect()

						// Log error.
						logger.Errorw("err", err)

					}
				} else {
					// We're disconnected, unset the channel.
					p.channelLock.Lock()
					p.channel = nil
					p.channelLock.Unlock()
				}
			}
		}
	}()

	return &p
}

// Close will close the closer chan that will stop the reconnection loop,
// once makes sure this can only happen once.
func (p *Publisher) Close() {
	p.once.Do(func() {
		close(p.closer)
	})
}

type FileEvent struct {
	EventID  string `json:"event_id"`
	FilePath string `json:"file_path"`
}

// PurchaseSucceeded will publish the event when a purchase has succeeded.
func (p *Publisher) FileCreated(ctx context.Context, eventID string, filePath string) error {
	payload := FileEvent{
		EventID:  eventID,
		FilePath: filePath,
	}

	fmt.Println("get file")
	bytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload for event ID = %s: %w", eventID, domain.ErrBadRequest)
	}

	return p.publish(ctx, "tsv.created", bytes)
}

// Publish will publish the message on the given exchange.
func (p *Publisher) publish(ctx context.Context, routingKey string, payload []byte) error {
	if p.channel == nil {
		return errors.New("no available amqp channel in publisher")
	}

	// Extract the tracing information from the ctx.

	return p.channel.PublishWithContext(ctx,
		p.exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			Headers:     nil,
			ContentType: "application/json",
			Body:        payload,
		})
}
