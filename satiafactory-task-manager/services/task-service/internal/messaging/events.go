package messaging

import "time"

const (
	EventCreated = "task.created"
	EventUpdated = "task.updated"
	EventDeleted = "task.deleted"
)

type TaskEvent struct {
	Type              string    `json:"type"`
	TaskID            int64     `json:"task_id"`
	UserID            int64     `json:"user_id,omitempty"`
	AssignedToUserIDs []int64   `json:"assigned_to_user_ids,omitempty"`
	Status            string    `json:"status,omitempty"`
	OccurredAt        time.Time `json:"occurred_at"`
}
