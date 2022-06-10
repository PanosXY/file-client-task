package main

import (
	"flag"
	"fmt"

	"github.com/PanosXY/file-client-task/client"
	"github.com/PanosXY/file-client-task/utils"
)

func main() {
	url := flag.String("url", "http://localhost:8080/", "The requested file server's url")
	char := flag.String("char", "A", "The files' matching character")
	debug := flag.Bool("debug", true, "Logger's debug flag")
	maxWorkers := flag.Uint("max-concurrent-downloads", 4, "The number of maximum concurrent downloads")
	flag.Parse()

	log, err := utils.NewLogger(*debug)
	if err != nil {
		return
	}
	defer log.Info("Client stopped")

	log.Info("Initializing...")
	c, err := client.NewClient(*url, *char, *maxWorkers, log)
	if err != nil {
		log.Error(fmt.Sprintf("Error initializing client: %v", err))
		return
	}
	if err := c.Do(); err != nil {
		log.Error(fmt.Sprintf("%v", err))
		return
	}
}
