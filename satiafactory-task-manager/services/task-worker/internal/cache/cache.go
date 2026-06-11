package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type TaskListCache struct {
	client  *redis.Client
	ttl     time.Duration
	enabled bool
}

func New(redisURL string, ttlSeconds int) *TaskListCache {
	if redisURL == "" {
		log.Println("REDIS_URL not set, cache warming disabled")
		return &TaskListCache{}
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Printf("invalid REDIS_URL: %v", err)
		return &TaskListCache{}
	}
	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("redis unavailable: %v", err)
		return &TaskListCache{}
	}
	if ttlSeconds <= 0 {
		ttlSeconds = 60
	}
	return &TaskListCache{client: client, ttl: time.Duration(ttlSeconds) * time.Second, enabled: true}
}

func listKey(scope string, userID int64) string {
	switch scope {
	case "mine":
		return fmt.Sprintf("tasks:list:mine:%d", userID)
	case "completed":
		return "tasks:list:completed"
	default:
		return "tasks:list:all"
	}
}

func (c *TaskListCache) Set(scope string, userID int64, body []byte) {
	if !c.enabled || len(body) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := c.client.Set(ctx, listKey(scope, userID), body, c.ttl).Err(); err != nil {
		log.Printf("redis set %s: %v", listKey(scope, userID), err)
	}
}

func (c *TaskListCache) InvalidateLists(userIDs ...int64) {
	if !c.enabled {
		return
	}
	keys := []string{listKey("all", 0), listKey("completed", 0)}
	seen := map[int64]bool{}
	for _, uid := range userIDs {
		if uid <= 0 || seen[uid] {
			continue
		}
		seen[uid] = true
		keys = append(keys, listKey("mine", uid))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		log.Printf("redis invalidate: %v", err)
	}
}
