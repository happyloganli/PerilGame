package main

import (
	"fmt"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"os"
	"os/signal"
)

func main() {

	// Connect to rabbitMQ
	fmt.Println("Starting Peril server...")
	amqpURL := "amqp://guest:guest@localhost:5672/"
	dial, err := amqp.Dial(amqpURL)
	if err != nil {
		return
	}
	defer dial.Close()
	log.Printf("Connected to %s", amqpURL)

	// Open a new channel
	channel, err := dial.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer channel.Close()

	if err != nil {
		log.Printf("Error publishing peril event: %v", err)
		return
	}

	for {
		// Get input from the user
		words := gamelogic.GetInput()

		// If the slice is empty, continue to the next iteration
		if len(words) == 0 {
			continue
		}

		// Handle the different commands based on the first word
		switch words[0] {
		case "pause":
			fmt.Println("Sending a pause message")
			err = pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
				IsPaused: true,
			})
			if err != nil {
				log.Fatalf("Failed to publish pause message: %v", err)
			}
		case "resume":
			fmt.Println("Sending a resume message")
			err = pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
				IsPaused: false,
			})
			if err != nil {
				log.Fatalf("Failed to publish resume message: %v", err)
			}
		case "quit":
			fmt.Println("Exiting...")
			break
		default:
			fmt.Println("I don't understand the command:", words[0])
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	log.Println("Shutting down...")
}
