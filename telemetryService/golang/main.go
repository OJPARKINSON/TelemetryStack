package main

import (
	"fmt"

	"github.com/ojparkinson/telemetryService/internal/config"
	"github.com/ojparkinson/telemetryService/internal/queue"
)

func main() {
	fmt.Println("Starting subscriber")

	config := config.NewConfig()

	messaging := queue.NewSubscriber()

	fmt.Println("Starting to consume")
	messaging.Subscribe(config)

}
