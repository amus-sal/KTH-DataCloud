package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amus-sal/kth-datacloud-arangoDB-converter/connection"
	"github.com/amus-sal/kth-datacloud-arangoDB-converter/event"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/amus-sal/kth-datacloud-arangoDB-converter/rabbitmq"
)

type serviceSecret struct {
	RabbitMQ rabbitSecret `json:"rabbitmq"`
}

type rabbitSecret struct {
	URL      string `json:"url"`
	Exchange string `json:"v2exchange"`
}

func main() {
	if len(os.Args) > 1 {
		file := os.Args[1]
		commandhandler.Handle(file)
		os.Exit(0)
	}
	const serviceName = "csv-converter-api"

	// Read the secret values defined in the service from Vault.
	var secret serviceSecret

	jsonFile, err := os.Open("./config.json")
	if err != nil {
		log.Fatal(err)
	}
	byteValue, _ := io.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, &secret)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(secret.RabbitMQ.URL)
	// Register Zap logger.
	zapConfig := zap.NewProductionConfig()
	zapConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)
	zapConfig.EncoderConfig.CallerKey = zapcore.OmitKey

	// Registers two zap cores as an option, one for stdout for info level logs,
	// and one for stderr for error level logs.
	opt := zap.WrapCore(func(zapcore.Core) zapcore.Core {
		return zapcore.NewTee(
			zapcore.NewCore(
				zapcore.NewJSONEncoder(zapConfig.EncoderConfig),
				zapcore.Lock(os.Stdout),
				zap.LevelEnablerFunc(func(level zapcore.Level) bool {
					return level == zapcore.InfoLevel || level == zapcore.DebugLevel
				}),
			),
			zapcore.NewCore(
				zapcore.NewJSONEncoder(zapConfig.EncoderConfig),
				zapcore.Lock(os.Stderr),
				zap.LevelEnablerFunc(func(level zapcore.Level) bool {
					return level == zapcore.ErrorLevel || level == zapcore.FatalLevel
				}),
			),
		)
	})

	logger, err := zapConfig.Build(opt)
	if err != nil {
		log.Fatal(err)
	}

	// Flushes buffer, if any.
	defer func() {
		if err := logger.Sync(); err != nil {
			log.Fatal(err)
		}
	}()

	// Event service receives async events and handles the corresponding business logic.
	eventService := event.NewService()

	//AMQP connector and consumer connected to RabbitMQ.
	amqpConnector2 := connection.NewConnector(
		secret.RabbitMQ.URL,
		connection.WithLogger(logger.Sugar()),
	)
	consumer := rabbitmq.NewConsumer(
		amqpConnector2,
		secret.RabbitMQ.Exchange,
		eventService,
	)

	// Channel to receive errors on from different go routines, such as the http server.
	errorChannel := make(chan error)

	// Start RabbitMQ consumer and send any error on the channel.
	go func() {
		errorChannel <- consumer.Start()
	}()

	// Capture interupts, to handle them gracefully.
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errorChannel <- fmt.Errorf("got terminating signal: %s", <-c)
	}()

	// Wait for errors on the error channel, this will stall until an error is received.
	if err := <-errorChannel; err != nil {
		log.Fatal(err)
	}
}

func envString(key string, fallback string) string {
	if value, ok := syscall.Getenv(key); ok {
		return value
	}
	return fallback
}
