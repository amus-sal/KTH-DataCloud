package event

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
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

	output, err := csvConverter(filePath)
	if err != nil {
		fmt.Println(err)
	}
	err = s.publisher.FileCreated(ctx, "123", output)
	if err != nil {
		fmt.Println(err)
	}
	return nil
}

func csvConverter(tsvFileSource string) (string, error) {
	// Open the TSV file for reading
	tsvFile, err := os.Open(tsvFileSource)
	if err != nil {
		return "", err
	}
	defer tsvFile.Close()

	// Create a CSV file for writing
	csvFile, err := os.Create("file.csv")
	if err != nil {
		return "", err
	}
	defer csvFile.Close()

	// Create CSV writer
	csvWriter := csv.NewWriter(csvFile)
	defer csvWriter.Flush()

	// Create a TSV reader
	tsvReader := bufio.NewReader(tsvFile)

	for {
		// Read a line from the TSV file
		line, err := tsvReader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			break
		}

		// Split the TSV line by tabs
		fields := strings.Split(strings.TrimSpace(line), "\t")

		// Write the fields to the CSV file
		err = csvWriter.Write(fields)
		if err != nil {
			return "", err
		}
	}
	return "./file.csv", nil
}
