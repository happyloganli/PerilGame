package gamelogic

import (
	"context"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"time"
)

type GameLog struct {
	CurrentTime time.Time
	Message     string
	Username    string
}

type GameLogWorker struct {
	client  *mongo.Client
	connect *amqp.Connection
}

func NewGameLogWorker(cl *mongo.Client, ct *amqp.Connection) *GameLogWorker {
	return &GameLogWorker{
		client:  cl,
		connect: ct,
	}
}

func (g *GameLogWorker) Start() {
	coll := g.client.Database("peril").Collection("game_logs")
	// Declare and bind the game log queue
	pubsub.DeclareAndBind(g.connect, routing.ExchangePerilTopic, routing.GameLogSlug, routing.GameLogSlug+".*", amqp.Transient)
	pubsub.SubscribeGob(g.connect, routing.ExchangePerilTopic, routing.GameLogSlug, routing.GameLogSlug+".*", amqp.Transient, handleGameLog(coll))
}

func handleGameLog(coll *mongo.Collection) func(GameLog) pubsub.AckType {
	return func(gl GameLog) pubsub.AckType {
		log.Printf("Received game log: %v", gl)
		coll.InsertOne(context.Background(), gl)
		return pubsub.Ack
	}
}
