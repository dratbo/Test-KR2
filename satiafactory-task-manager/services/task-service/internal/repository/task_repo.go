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
                           COALESCE(t.target_item_class_name, ''), COALESCE(t.target_amount, 0)
                    FROM tasks t
                    LEFT JOIN users u1 ON t.user_id = u1.id
                    LEFT JOIN users u2 ON t.assigned_to_user_id = u2.id`

func scanTask(row interface{ Scan(...any) error }) (*models.Task, error) {
	var t models.Task
	var assignedID sql.NullInt64
	err := row.Scan(&t.ID, &t.UserID, &t.CreatorName, &assignedID, &t.AssigneeName,
		&t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.TargetItemClassName, &t.TargetAmount)
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
	query := `INSERT INTO tasks (user_id, title, description, status, created_at, target_item_class_name, target_amount, assigned_to_user_id)
              VALUES ($1, $2, $3, $4, NOW(), $5, $6, $7) RETURNING id, created_at`
	err := r.db.QueryRow(query, task.UserID, task.Title, task.Description, task.Status,
		task.TargetItemClassName, task.TargetAmount, task.AssignedToUserID).Scan(&task.ID, &task.CreatedAt)
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

func (r *TaskRepository) Update(id int64, status *string, assignedTo *int64) (*models.Task, error) {
	current, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	newStatus := current.Status
	if status != nil && *status != "" {
		newStatus = *status
	}
	newAssigned := current.AssignedToUserID
	if assignedTo != nil {
		if *assignedTo == 0 {
			newAssigned = nil
		} else {
			newAssigned = assignedTo
		}
	}
	_, err = r.db.Exec(`UPDATE tasks SET status = $1, assigned_to_user_id = $2 WHERE id = $3`,
		newStatus, newAssigned, id)
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
