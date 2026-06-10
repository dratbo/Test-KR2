package models

import "time"

type Task struct {
	ID                  int64     `json:"id"`
	UserID              int64     `json:"user_id"`
	CreatorName         string    `json:"creator_name,omitempty"`
	AssignedToUserID    *int64    `json:"assigned_to_user_id,omitempty"`
	AssigneeName        string    `json:"assignee_name,omitempty"`
	Title               string    `json:"title"`
	Description         string    `json:"description,omitempty"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	TargetItemClassName string    `json:"target_item_class_name,omitempty"`
	TargetAmount        float64   `json:"target_amount,omitempty"`
}

type CreateTaskRequest struct {
	Title               string  `json:"title"`
	Description         string  `json:"description"`
	TargetItemClassName string  `json:"target_item_class_name"`
	TargetAmount        float64 `json:"target_amount"`
	AssignedToUserID    *int64  `json:"assigned_to_user_id,omitempty"`
}

type UpdateTaskRequest struct {
	Status           *string `json:"status,omitempty"`
	AssignedToUserID *int64  `json:"assigned_to_user_id,omitempty"`
}
