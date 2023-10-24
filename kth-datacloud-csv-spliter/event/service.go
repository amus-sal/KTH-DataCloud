package event

import (
	"context"
	"fmt"

	splitCsv "github.com/tolik505/split-csv"
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

	files := splitCsvFile(filePath)
	for _, file := range files {
		err := s.publisher.FileCreated(ctx, "123", file)
		if err != nil {
			fmt.Println(err)
		}
	}

	return nil
}

func splitCsvFile(file string) []string {
	splitter := splitCsv.New()
	splitter.Separator = ";"     // "," is by default
	splitter.FileChunkSize = 100 //in bytes (100MB)
	result, _ := splitter.Split(file, ".")
	return result
}
