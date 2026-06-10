package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/dratbo/satisfactory-task-manager/task-service/internal/models"
)

var ErrTaskNotFound = errors.New("task not found")

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
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

func scanTask(row interface{ Scan(...any) error }) (*models.Task, error) {
	var t models.Task
	var assignedID sql.NullInt64
	err := row.Scan(&t.ID, &t.UserID, &t.CreatorName, &assignedID, &t.AssigneeName,
		&t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.TargetItemClassName, &t.TargetAmount, &t.HubTier,
		&t.ProductionShards, &t.ConveyorMk, &t.PipeMk)
	if err != nil {
		return nil, err
	}
	if assignedID.Valid {
		id := assignedID.Int64
		t.AssignedToUserID = &id
	}
	return &t, nil
}

func (r *TaskRepository) Create(task *models.Task) error {
	hubTier := task.HubTier
	if hubTier <= 0 {
		hubTier = 9
	}
	query := `INSERT INTO tasks (user_id, title, description, status, created_at, target_item_class_name, target_amount, hub_tier, assigned_to_user_id)
              VALUES ($1, $2, $3, $4, NOW(), $5, $6, $7, $8) RETURNING id, created_at`
	err := r.db.QueryRow(query, task.UserID, task.Title, task.Description, task.Status,
		task.TargetItemClassName, task.TargetAmount, hubTier, task.AssignedToUserID).Scan(&task.ID, &task.CreatedAt)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	created, err := r.GetByID(task.ID)
	if err != nil {
		return err
	}
	*task = *created
	return nil
}

func (r *TaskRepository) GetAssignedTo(userID int64) ([]models.Task, error) {
	rows, err := r.db.Query(taskSelect+` WHERE t.assigned_to_user_id = $1 AND t.status <> 'completed' ORDER BY t.created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("query my tasks: %w", err)
	}
	defer rows.Close()
	var tasks []models.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, *t)
	}
	return tasks, nil
}

func (r *TaskRepository) GetAll() ([]models.Task, error) {
	rows, err := r.db.Query(taskSelect + ` WHERE t.status <> 'completed' ORDER BY t.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, *t)
	}
	return tasks, nil
}

func (r *TaskRepository) GetCompleted() ([]models.Task, error) {
	rows, err := r.db.Query(taskSelect + ` WHERE t.status = 'completed' ORDER BY t.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query completed tasks: %w", err)
	}
	defer rows.Close()
	var tasks []models.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, *t)
	}
	return tasks, nil
}

func (r *TaskRepository) GetByID(id int64) (*models.Task, error) {
	row := r.db.QueryRow(taskSelect+` WHERE t.id = $1`, id)
	t, err := scanTask(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("get task: %w", err)
	}
	return t, nil
}

func (r *TaskRepository) Update(id int64, req models.UpdateTaskRequest) (*models.Task, error) {
	current, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	newStatus := current.Status
	if req.Status != nil && *req.Status != "" {
		newStatus = *req.Status
	}
	newAssigned := current.AssignedToUserID
	if req.AssignedToUserID != nil {
		if *req.AssignedToUserID == 0 {
			newAssigned = nil
		} else {
			newAssigned = req.AssignedToUserID
		}
	}
	newHubTier := current.HubTier
	if req.HubTier != nil && *req.HubTier > 0 {
		newHubTier = *req.HubTier
	}
	newShards := current.ProductionShards
	if req.ProductionShards != nil {
		newShards = *req.ProductionShards
		if newShards < 0 {
			newShards = 0
		}
	}
	newConveyorMk := current.ConveyorMk
	if req.ConveyorMk != nil {
		newConveyorMk = *req.ConveyorMk
		if newConveyorMk < 0 {
			newConveyorMk = 0
		}
	}
	newPipeMk := current.PipeMk
	if req.PipeMk != nil {
		newPipeMk = *req.PipeMk
		if newPipeMk < 0 {
			newPipeMk = 0
		}
	}
	_, err = r.db.Exec(`UPDATE tasks SET status = $1, assigned_to_user_id = $2, hub_tier = $3,
		production_shards = $4, conveyor_mk = $5, pipe_mk = $6 WHERE id = $7`,
		newStatus, newAssigned, newHubTier, newShards, newConveyorMk, newPipeMk, id)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	return r.GetByID(id)
}

func (r *TaskRepository) DeleteByID(id int64) error {
	result, err := r.db.Exec(`DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrTaskNotFound
	}
	return nil
}
