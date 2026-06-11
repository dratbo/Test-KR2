package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

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
	HubTier             int       `json:"hub_tier,omitempty"`
	ProductionShards    int       `json:"production_shards,omitempty"`
	ConveyorMk          int       `json:"conveyor_mk,omitempty"`
	PipeMk              int       `json:"pipe_mk,omitempty"`
}

type TaskRepository struct {
	db *sql.DB
}

func New(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

const taskSelect = `SELECT t.id, t.user_id, COALESCE(u1.username, ''),
                           t.assigned_to_user_id, COALESCE(u2.username, ''),
                           t.title, t.description, t.status, t.created_at,
                           COALESCE(t.target_item_class_name, ''), COALESCE(t.target_amount, 0),
                           COALESCE(t.hub_tier, 9),
                           COALESCE(t.production_shards, 0), COALESCE(t.conveyor_mk, 0), COALESCE(t.pipe_mk, 0)
                    FROM tasks t
                    LEFT JOIN users u1 ON t.user_id = u1.id
                    LEFT JOIN users u2 ON t.assigned_to_user_id = u2.id`

func (r *TaskRepository) ListAllJSON() ([]byte, error) {
	return r.listJSON(taskSelect + ` ORDER BY t.created_at DESC`)
}

func (r *TaskRepository) ListCompletedJSON() ([]byte, error) {
	return r.listJSON(taskSelect + ` WHERE t.status = 'completed' ORDER BY t.created_at DESC`)
}

func (r *TaskRepository) ListMineJSON(userID int64) ([]byte, error) {
	return r.listJSON(taskSelect+` WHERE t.assigned_to_user_id = $1 AND t.status <> 'completed' ORDER BY t.created_at DESC`, userID)
}

func (r *TaskRepository) listJSON(query string, args ...any) ([]byte, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var assignedID sql.NullInt64
		if err := rows.Scan(&t.ID, &t.UserID, &t.CreatorName, &assignedID, &t.AssigneeName,
			&t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.TargetItemClassName, &t.TargetAmount, &t.HubTier,
			&t.ProductionShards, &t.ConveyorMk, &t.PipeMk); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		if assignedID.Valid {
			id := assignedID.Int64
			t.AssignedToUserID = &id
		}
		tasks = append(tasks, t)
	}
	if tasks == nil {
		tasks = []Task{}
	}
	return json.Marshal(tasks)
}
