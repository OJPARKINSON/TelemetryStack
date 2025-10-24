package main

import (
	"fmt"

	"github.com/ojparkinson/telemetryService/internal/config"
	"github.com/ojparkinson/telemetryService/internal/persistance"
	"github.com/ojparkinson/telemetryService/internal/queue"
)

func main() {
	fmt.Println("Starting subscriber")

	config := config.NewConfig()

	senderPool, err := persistance.NewSenderPool(10, "localhost", 9000)
	if err != nil {
		fmt.Println("Failed to create sender pool: ", err)
	}
	messaging := queue.NewSubscriber(senderPool)

	fmt.Println("Starting to consume")
	messaging.Subscribe(config)
}
