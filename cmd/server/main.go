package main

import (
	"context"
	"fmt"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

func main() {

	// Connect to rabbitMQ
	fmt.Println("Starting Peril server...")
	amqpURL := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return
	}
	defer conn.Close()
	log.Printf("Connected to %s", amqpURL)

	// Open a new channel
	channel, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer channel.Close()

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())

	gameLogWorker := gamelogic.NewGameLogWorker(client, conn)
	go gameLogWorker.Start()

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

	log.Println("Shutting down...")
}
