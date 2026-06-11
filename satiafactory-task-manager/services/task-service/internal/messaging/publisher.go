package messaging

import (
	"context"
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/dratbo/satisfactory-task-manager/task-service/internal/metrics"
)

const exchangeName = "task.events"

type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	enabled bool
}

func NewPublisher(rabbitURL string) *Publisher {
	if rabbitURL == "" {
		log.Println("RABBITMQ_URL not set, task event publishing disabled")
		return &Publisher{}
	}

	conn, err := dialWithRetry(rabbitURL, 10)
	if err != nil {
		log.Printf("rabbitmq unavailable, publishing disabled: %v", err)
		return &Publisher{}
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		log.Printf("rabbitmq channel error, publishing disabled: %v", err)
		return &Publisher{}
	}

	if err := ch.ExchangeDeclare(exchangeName, "fanout", true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		log.Printf("rabbitmq exchange declare failed, publishing disabled: %v", err)
		return &Publisher{}
	}

	log.Printf("rabbitmq publisher enabled (exchange=%s)", exchangeName)
	return &Publisher{conn: conn, channel: ch, enabled: true}
}

func (p *Publisher) Close() {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}

func (p *Publisher) Publish(event TaskEvent) {
	if !p.enabled {
		return
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}

	body, err := json.Marshal(event)
	if err != nil {
		log.Printf("marshal task event: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = p.channel.PublishWithContext(ctx, exchangeName, "", false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
	if err != nil {
		log.Printf("publish task event %s: %v", event.Type, err)
		return
	}
	metrics.RecordEventPublished(event.Type)
}

func dialWithRetry(url string, attempts int) (*amqp.Connection, error) {
	var last error
	for i := 0; i < attempts; i++ {
		conn, err := amqp.Dial(url)
		if err == nil {
			return conn, nil
		}
		last = err
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	return nil, last
}
