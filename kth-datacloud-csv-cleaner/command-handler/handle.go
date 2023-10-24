package commandhandler

import (
	"log"
	"os"
	"time"
)

func Handle(path string) {

	f, err := os.Create("/tmp/output.txt")

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	time.Sleep(5 * time.Second)
	_, err = f.WriteString(path)

}
