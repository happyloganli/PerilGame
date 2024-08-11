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
	"strings"
)

func main() {
	fmt.Println("Starting Peril client...")
	// Connect to rabbitMQ
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

	// Welcome the client
	username, _ := gamelogic.ClientWelcome()
	gameState := gamelogic.NewGameState(username)

	// Declare and bind the pausing queue
	pubsub.DeclareAndBind(dial, routing.ExchangePerilDirect, routing.PauseKey+"."+username, routing.PauseKey, amqp.Transient)
	// Each client subscribes pausing messages from the pausing queue
	pubsub.SubscribeJSON(dial, routing.ExchangePerilDirect, routing.PauseKey+"."+username, routing.PauseKey, amqp.Transient, handlerPause(gameState))

	// Declare and bing the moving queue
	pubsub.DeclareAndBind(dial, routing.ExchangePerilTopic, routing.ArmyMovesPrefix+"."+username, routing.ArmyMovesPrefix+".*", amqp.Transient)
	// Each client subscribes moving messages from the moving queue
	pubsub.SubscribeJSON(dial, routing.ExchangePerilTopic, routing.ArmyMovesPrefix+"."+username, routing.ArmyMovesPrefix+".*", amqp.Transient, handlerMove(gameState, channel, username))

	// Declare and bind the war queue
	pubsub.DeclareAndBind(dial, routing.ExchangePerilTopic, routing.WarRecognitionsPrefix+"."+username, routing.WarRecognitionsPrefix+".*", amqp.Transient)
	// Each client subscribes war messages from the war queue
	pubsub.SubscribeJSON(dial, routing.ExchangePerilTopic, routing.WarRecognitionsPrefix+"."+username, routing.WarRecognitionsPrefix+".*", amqp.Transient, handlerWar(gameState))

	// Wait for user's input command
	for {
		commands := gamelogic.GetInput()
		command := strings.ToLower(commands[0])
		switch command {
		case "move":
			armyMove, _ := gameState.CommandMove(commands)
			pubsub.PublishJSON(channel, routing.ExchangePerilTopic, routing.ArmyMovesPrefix+"."+username, armyMove)
			log.Printf("Successfully published: %s", command)
		case "spawn":
			gameState.CommandSpawn(commands)
		case "status":
			gameState.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			log.Printf("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
		default:
			log.Printf("Unknown command: %s", command)
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	log.Println("Shutting down...")
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		// Defer a print statement to give the user a new prompt
		defer fmt.Print("> ")

		// Use the game state's HandlePause method to pause the game
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState, channel *amqp.Channel, username string) func(move gamelogic.ArmyMove) pubsub.AckType {
	return func(am gamelogic.ArmyMove) pubsub.AckType {
		// Defer a print statement to give the user a new prompt
		defer fmt.Print("> ")

		// Use the game state's HandlePause method to pause the game
		moveOutcome := gs.HandleMove(am)
		switch moveOutcome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			err := pubsub.PublishJSON(channel, routing.ExchangePerilTopic, routing.WarRecognitionsPrefix+"."+username, gamelogic.RecognitionOfWar{
				Attacker: am.Player,
				Defender: gs.Player,
			})
			if err != nil {
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard
		default:
			return pubsub.NackDiscard
		}
	}
}

func handlerWar(gs *gamelogic.GameState) func(war gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(rw gamelogic.RecognitionOfWar) pubsub.AckType {
		// Defer a print statement to give the user a new prompt
		defer fmt.Print("> ")

		// Use the game state's HandleWar method to pause the game
		outcome, _, _ := gs.HandleWar(rw)
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.Ack
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			return pubsub.Ack
		case gamelogic.WarOutcomeYouWon:
			return pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			return pubsub.Ack
		default:
			log.Printf("Unknown outcome: %s", outcome)
			return pubsub.NackDiscard
		}
		return pubsub.Ack
	}
}
