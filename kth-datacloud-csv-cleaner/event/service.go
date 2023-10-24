package event

import (
	"context"
	"fmt"
	"time"
)

type publisher interface {
	FileCreated(ctx context.Context, eventID string, filePath string) error
}

type Service struct {
	publisher publisher
}

// NewService will return a new service with all dependencies.
func NewService(publisher publisher) Service {
	return Service{
		publisher: publisher,
	}
}

// Handle will handle all incoming events.
func (s Service) Handle(ctx context.Context, eventID string, filePath string) error {

	fmt.Println("received event")
	time.Sleep(5 * time.Second)

	err := s.publisher.FileCreated(ctx, "123", filePath)
	if err != nil {
		fmt.Println(err)
	}
	return nil
}
