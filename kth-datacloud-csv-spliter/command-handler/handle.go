package commandhandler

import (
	"fmt"
	"log"
	"os"
)

func Handle(path string) {

	dests, err := splitCsvFile(path)
	if err != nil {
		fmt.Println(err)
	}
	f, err := os.Create("/tmp/output.txt")

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err = f.WriteString(dests)

	if err != nil {
		log.Fatal(err)
	}
}
func splitCsvFile(file string) []string {
	splitter := splitCsv.New()
	splitter.Separator = ";"     // "," is by default
	splitter.FileChunkSize = 100 //in bytes (100MB)
	result, _ := splitter.Split(file, ".")
	return result
}
