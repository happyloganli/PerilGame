package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
)

type AckType int

const (
	// Ack indicates the message has been successfully processed
	Ack AckType = iota
	// NackRequeue indicates the message should be requeued
	NackRequeue
	// NackDiscard indicates the message should be discarded
	NackDiscard
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	body, _ := json.Marshal(val)
	err := ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
	if err != nil {
		log.Printf("Error publishing message to exchange: %v", err)
		return err
	}
	return nil
}

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(val)
	body := buffer.Bytes()
	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "application/gob",
		Body:        body,
	})
	log.Printf("Published message to exchange: %v", val)
	if err != nil {
		log.Printf("Error publishing message to exchange: %v", err)
		return err
	}
	return nil
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType uint8, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	var queue amqp.Queue

	if simpleQueueType == amqp.Transient {
		queue, err = ch.QueueDeclare(queueName, false, true, true, false, amqp.Table{"x-dead-letter-exchange": "peril_dlx"})
	} else {
		queue, err = ch.QueueDeclare(queueName, true, false, false, false, amqp.Table{"x-dead-letter-exchange": "peril_dlx"})
	}

	err = ch.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	return ch, queue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType uint8,
	handler func(T) AckType,
) error {
	DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	ch, _ := conn.Channel()
	deliveries, _ := ch.Consume(queueName, "", false, false, false, false, nil)
	go func() {
		for delivery := range deliveries {
			var msg T
			json.Unmarshal(delivery.Body, &msg)
			ackType := handler(msg)
			switch ackType {
			case Ack:
				delivery.Ack(false)
			case NackRequeue:
				delivery.Nack(false, true)
			case NackDiscard:
				delivery.Nack(false, false)
			}

		}
	}()
	return nil
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType uint8,
	handler func(T) AckType,
) error {
	//DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	ch, _ := conn.Channel()
	deliveries, _ := ch.Consume(queueName, "", false, false, false, false, nil)
	go func() {
		for delivery := range deliveries {
			var msg T
			decoder := gob.NewDecoder(bytes.NewReader(delivery.Body))
			decoder.Decode(&msg)
			ackType := handler(msg)
			switch ackType {
			case Ack:
				delivery.Ack(false)
			case NackRequeue:
				delivery.Nack(false, true)
			case NackDiscard:
				delivery.Nack(false, false)
			}
		}
	}()
	return nil
}
