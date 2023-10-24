package event

import (
	"context"
	"fmt"
	"time"
)

type Service struct {
}

// NewService will return a new service with all dependencies.
func NewService() Service {
	return Service{}
}

// Handle will handle all incoming events.
func (s Service) Handle(ctx context.Context, eventID string, filePath string) error {

	fmt.Println("received event")

	time.Sleep(5 * time.Second)
	return nil
}
