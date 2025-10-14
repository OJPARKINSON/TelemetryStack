package main

import (
	"fmt"

	"github.com/ojparkinson/telemetryService/internal"
)

func main() {
	fmt.Println("hello")

	messaging := internal.NewSubscriber()

	messaging.Subscribe()

	fmt.Println("bonk")
}
