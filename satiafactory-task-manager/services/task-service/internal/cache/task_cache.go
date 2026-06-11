package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// TaskListCache stores JSON responses for GET /tasks list endpoints.
type TaskListCache struct {
	client  *redis.Client
	ttl     time.Duration
	enabled bool
}

func NewTaskListCache(redisURL string, ttlSeconds int) *TaskListCache {
	if redisURL == "" {
		log.Println("REDIS_URL not set, task list caching disabled")
		return &TaskListCache{}
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Printf("invalid REDIS_URL, caching disabled: %v", err)
		return &TaskListCache{}
	}
	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("redis unavailable, caching disabled: %v", err)
		return &TaskListCache{}
	}
	if ttlSeconds <= 0 {
		ttlSeconds = 60
	}
	log.Printf("redis task list cache enabled (ttl=%ds)", ttlSeconds)
	return &TaskListCache{
		client:  client,
		ttl:     time.Duration(ttlSeconds) * time.Second,
		enabled: true,
	}
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

// Get returns cached JSON for a task list scope.
func (c *TaskListCache) Get(scope string, userID int64) ([]byte, bool) {
	if !c.enabled {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	body, err := c.client.Get(ctx, listKey(scope, userID)).Bytes()
	if err != nil {
		return nil, false
	}
	return body, true
}

// Set stores JSON for a task list scope.
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

// InvalidateLists drops shared list keys and per-user "mine" lists.
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
