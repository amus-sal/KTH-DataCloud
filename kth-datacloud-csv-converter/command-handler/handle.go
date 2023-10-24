package commandhandler

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
)



func Handle(path string) {

	dest, err := csvConverter(path)
	if err != nil {
		fmt.Println(er r)
	}
	f, err := os.Create("/tmp/output.txt")

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err = f.WriteString(dest)

	if err != nil {
		log.Fatal(err)
	}
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
