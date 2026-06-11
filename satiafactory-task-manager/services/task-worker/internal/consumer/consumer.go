package consumer

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/dratbo/satisfactory-task-manager/task-worker/internal/cache"
	"github.com/dratbo/satisfactory-task-manager/task-worker/internal/events"
	"github.com/dratbo/satisfactory-task-manager/task-worker/internal/repository"
)

const (
	exchangeName = "task.events"
	queueName    = "task.worker"
)

type Consumer struct {
	repo  *repository.TaskRepository
	cache *cache.TaskListCache
}

func New(repo *repository.TaskRepository, taskCache *cache.TaskListCache) *Consumer {
	return &Consumer{repo: repo, cache: taskCache}
}

func (c *Consumer) Run(rabbitURL string) error {
	conn, err := dialWithRetry(rabbitURL, 15)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(exchangeName, "fanout", true, false, false, false, nil); err != nil {
		return err
	}
	q, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return err
	}
	if err := ch.QueueBind(q.Name, "", exchangeName, false, nil); err != nil {
		return err
	}
	if err := ch.Qos(1, 0, false); err != nil {
		return err
	}

	deliveries, err := ch.Consume(q.Name, "task-worker", false, false, false, false, nil)
	if err != nil {
		return err
	}

	log.Printf("task-worker listening on queue %s", queueName)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-sig:
			log.Println("task-worker shutting down")
			return nil
		case d, ok := <-deliveries:
			if !ok {
				return nil
			}
			c.handle(d)
			d.Ack(false)
		}
	}
}

func (c *Consumer) handle(d amqp.Delivery) {
	var event events.TaskEvent
	if err := json.Unmarshal(d.Body, &event); err != nil {
		log.Printf("invalid event payload: %v", err)
		return
	}

	log.Printf("[audit] %s task_id=%d user_id=%d assignees=%v status=%s",
		event.Type, event.TaskID, event.UserID, event.AssignedToUserIDs, event.Status)

	c.cache.InvalidateLists(event.AssignedToUserIDs...)
	c.warmLists(event.AssignedToUserIDs)
}

func (c *Consumer) warmLists(userIDs []int64) {
	if body, err := c.repo.ListAllJSON(); err == nil {
		c.cache.Set("all", 0, body)
	} else {
		log.Printf("warm all list: %v", err)
	}
	if body, err := c.repo.ListCompletedJSON(); err == nil {
		c.cache.Set("completed", 0, body)
	} else {
		log.Printf("warm completed list: %v", err)
	}
	seen := map[int64]bool{}
	for _, uid := range userIDs {
		if uid <= 0 || seen[uid] {
			continue
		}
		seen[uid] = true
		if body, err := c.repo.ListMineJSON(uid); err == nil {
			c.cache.Set("mine", uid, body)
		} else {
			log.Printf("warm mine list user=%d: %v", uid, err)
		}
	}
}

func dialWithRetry(url string, attempts int) (*amqp.Connection, error) {
	var last error
	for i := 0; i < attempts; i++ {
		conn, err := amqp.Dial(url)
		if err == nil {
			return conn, nil
		}
		last = err
		log.Printf("rabbitmq dial attempt %d/%d: %v", i+1, attempts, err)
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	return nil, last
}
