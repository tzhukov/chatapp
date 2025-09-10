package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
	"src/config"
	"src/models"
)

func Writer(msg models.Message) {
	log.Printf("writing message to kafka topic %s", config.Topic)
	w := &kafka.Writer{
		Addr:     kafka.TCP(config.KafkaBroker),
		Topic:    config.Topic,
		Balancer: &kafka.LeastBytes{},
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("error marshalling message: %v", err)
		return
	}

	err = w.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(msg.MessageID),
			Value: msgBytes,
		},
	)
	if err != nil {
		log.Printf("failed to write messages to kafka: %v", err)
	} else {
		log.Printf("successfully wrote message to kafka: %s", msg.MessageID)
	}

	if err := w.Close(); err != nil {
		log.Printf("failed to close kafka writer: %v", err)
	}
}

func Reader(broadcast chan<- models.Message) {
	log.Printf("starting kafka reader on topic %s", config.Topic)
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{config.KafkaBroker},
		Topic:     config.Topic,
		Partition: 0,
		MinBytes:  10e3, // 10KB
		MaxBytes:  10e6, // 10MB
	})

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Printf("error reading message from kafka: %v", err)
			break
		}
		log.Printf("message read from kafka partition %d at offset %d", m.Partition, m.Offset)

		var msg models.Message
		err = json.Unmarshal(m.Value, &msg)
		if err != nil {
			log.Printf("error unmarshalling message from kafka: %v", err)
			continue
		}

		broadcast <- msg
	}

	if err := r.Close(); err != nil {
		log.Printf("failed to close kafka reader: %v", err)
	}
}
