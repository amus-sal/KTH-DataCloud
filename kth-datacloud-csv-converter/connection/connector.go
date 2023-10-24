package connection

import (
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type logger interface {
	Infow(msg string, keysAndValues ...interface{})
}

type connector struct {
	// url is the connection string for RabbitMQ.
	url string

	// conn is the current connection used by listeners.
	conn *amqp.Connection

	// connLock is a locking mechanism for reading or
	// writing the connection on different go routines.
	connLock sync.Mutex

	// reconnectDelay is the duration between each reconnection attempt.
	reconnectDelay time.Duration

	// listeners is a slice of consumers who are interested in when there
	// is a new connection available.
	listeners []chan<- *amqp.Connection

	// listenerLock is used when adding listeners to the list.
	listenerLock sync.Mutex

	// closer is used if the caller wants to close the entire connector down.
	closer chan struct{}

	// once is used to make sure it is only used exactly one time,
	// it's used for closing the connector.
	once sync.Once

	logger logger
}

// WithLogger adds logging to the connector.
func WithLogger(logger logger) func(*connector) {
	return func(c *connector) {
		c.logger = logger
	}
}

// NewConnector returns a pointer to an AMQP connector, it has to be a
// pointer because of a connection lock.
func NewConnector(url string, options ...func(*connector)) *connector {
	connector := connector{
		url:            url,
		closer:         make(chan struct{}),
		reconnectDelay: 5 * time.Second,
	}

	// Set options.
	for _, o := range options {
		o(&connector)
	}

	// Dial loop.
	go func() {
		// No sleep first iteration.
		var sleep time.Duration
		for {
			time.Sleep(sleep)
			sleep = connector.reconnectDelay

			connector.log("amqp: dialing...")

			conn, err := amqp.Dial(url)
			if err != nil {
				connector.log("amqp: could not dial", "err", err)
				continue
			}

			connector.log("amqp: successfully connected")

			// Setup close channel, to listen for closing events from the connection.
			closer := make(chan *amqp.Error)
			conn.NotifyClose(closer)

			// Notify listeners of the new connection available.
			connector.notify(conn)

			select {
			case err := <-closer:
				// Connection was closed, reconnect.
				connector.log("amqp: connection was closed", "err", err)

			case <-connector.closer:
				// Connector was closed, exit dial loop entirely.
				return
			}
		}
	}()

	return &connector
}

func (c *connector) NotifyConnection() <-chan *amqp.Connection {
	// Create a new listener channel.
	listener := make(chan *amqp.Connection)

	c.listenerLock.Lock()
	defer c.listenerLock.Unlock()

	// Append the new listener to our list.
	c.listeners = append(c.listeners, listener)

	// Read the current connection and send it on the channel
	// to give the consumer the connection immediately.
	go func() {
		c.connLock.Lock()
		defer c.connLock.Unlock()
		if c.conn != nil {
			select {
			case listener <- c.conn:
			case <-time.After(10 * time.Second):
			}
		}
	}()

	// Return channel to be listened on by consumer.
	return listener
}

// Reconnect will close the current connection which will
// trigger the close event and trigger a reconnection attempt.
func (c *connector) Reconnect() {
	c.connLock.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connLock.Unlock()
}

// Close will close the underlying amqp connection and notify
// all listeners that we have closed.
func (c *connector) Close() {
	c.once.Do(func() {
		// Close reconnection loop.
		close(c.closer)

		// Notify all listeners that we are closing by
		// closing the channels they are listening on.
		c.listenerLock.Lock()
		for _, l := range c.listeners {
			close(l)
		}

		c.listeners = nil
		c.listenerLock.Unlock()

		// Now finally close the amqp connection.
		c.connLock.Lock()
		if c.conn != nil {
			c.conn.Close()
		}
		c.connLock.Unlock()
	})
}

func (c *connector) log(msg string, keyAndValues ...interface{}) {
	if c.logger != nil {
		c.logger.Infow(msg, keyAndValues...)
	}
}

func (c *connector) notify(connection *amqp.Connection) {
	c.connLock.Lock()
	defer c.connLock.Unlock()

	c.listenerLock.Lock()
	defer c.listenerLock.Unlock()

	c.conn = connection

	if len(c.listeners) > 0 {
		for _, l := range c.listeners {
			l <- c.conn
		}
	}
}
