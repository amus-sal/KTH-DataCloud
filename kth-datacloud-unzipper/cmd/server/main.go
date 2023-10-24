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

	"github.com/amus-sal/kth-datacloud-unzipper/commandhandler"
	"github.com/amus-sal/kth-datacloud-unzipper/connection"
	"github.com/amus-sal/kth-datacloud-unzipper/httpencoder"
	"github.com/amus-sal/kth-datacloud-unzipper/httpserver"
	"github.com/amus-sal/kth-datacloud-unzipper/httpserver/unzipper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/amus-sal/kth-datacloud-unzipper/rabbitmq"
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
		os.exit(0)
	}

	var (
		httpAddress = envString("HTTP_ADDRESS", ":8085")
	)

	const serviceName = "unzipper-api"

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

	// Init RabbitMQ Publisher.
	amqpConnector := connection.NewConnector(secret.RabbitMQ.URL, connection.WithLogger(logger.Sugar()))
	publisher := rabbitmq.NewPublisher(amqpConnector, secret.RabbitMQ.Exchange, logger.Sugar())

	// Encoder to encode all http responses and errors.
	encoder := httpencoder.NewEncoder()

	// Handler that will handle all HTTP reqeusts on the / routes.
	unzipper := unzipper.NewHandler(encoder, publisher)

	// Channel to receive errors on from different go routines, such as the http server.
	errorChannel := make(chan error)

	// Open HTTP server and send it on the error channel, will stall until any error is returned.
	go func() {
		server := httpserver.NewServer(serviceName, httpAddress, 5*time.Second, unzipper, logger)
		errorChannel <- server.Open()
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
