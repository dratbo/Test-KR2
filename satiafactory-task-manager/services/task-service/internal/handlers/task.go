package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/dratbo/satisfactory-task-manager/task-service/internal/cache"
	"github.com/dratbo/satisfactory-task-manager/task-service/internal/metrics"
	"github.com/dratbo/satisfactory-task-manager/task-service/internal/messaging"
	"github.com/dratbo/satisfactory-task-manager/task-service/internal/middleware"
	"github.com/dratbo/satisfactory-task-manager/task-service/internal/models"
	"github.com/dratbo/satisfactory-task-manager/task-service/internal/repository"
)

type TaskHandler struct {
	repo      *repository.TaskRepository
	cache     *cache.TaskListCache
	publisher *messaging.Publisher
}

func NewTaskHandler(repo *repository.TaskRepository, taskCache *cache.TaskListCache, publisher *messaging.Publisher) *TaskHandler {
	return &TaskHandler{repo: repo, cache: taskCache, publisher: publisher}
}

func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	hubTier := req.HubTier
	if hubTier <= 0 {
		hubTier = 9
	}
	task := &models.Task{
		UserID:              userID,
		Title:               req.Title,
		Description:         req.Description,
		Status:              "pending",
		TargetItemClassName: req.TargetItemClassName,
		TargetAmount:        req.TargetAmount,
		HubTier:             hubTier,
		AssignedToUserID:    req.AssignedToUserID,
	}

	if err := h.repo.Create(task); err != nil {
		log.Printf("create task error: %v", err)
		http.Error(w, "failed to create task", http.StatusInternalServerError)
		return
	}

	h.invalidateAfterChange(task.AssignedToUserID)
	h.publishEvent(messaging.EventCreated, task.ID, task.UserID, task.Status, task.AssignedToUserID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) GetTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	scope := normalizeScope(r.URL.Query().Get("scope"))
	if body, hit := h.cache.Get(scope, userID); hit {
		metrics.RecordCacheHit(scope)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write(body)
		return
	}
	metrics.RecordCacheMiss(scope)

	var tasks []models.Task
	var err error
	switch scope {
	case "mine":
		tasks, err = h.repo.GetAssignedTo(userID)
	case "completed":
		tasks, err = h.repo.GetCompleted()
	default:
		tasks, err = h.repo.GetAll()
	}
	if err != nil {
		http.Error(w, "failed to get tasks", http.StatusInternalServerError)
		return
	}

	body, err := json.Marshal(tasks)
	if err != nil {
		http.Error(w, "failed to encode tasks", http.StatusInternalServerError)
		return
	}
	h.cache.Set(scope, userID, body)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(body)
}

func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	if _, ok := r.Context().Value(middleware.UserIDKey).(int64); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := parseTaskID(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}

	task, err := h.repo.GetByID(id)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := parseTaskID(strings.TrimSuffix(r.URL.Path, "/assign"))
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}

	before, err := h.repo.GetByID(id)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get task", http.StatusInternalServerError)
		return
	}

	var req models.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// «Взять задачу» — назначить на текущего пользователя
	if strings.HasSuffix(r.URL.Path, "/take") {
		req.AssignedToUserID = &userID
		if req.Status == nil {
			s := "in_progress"
			req.Status = &s
		}
	}

	task, err := h.repo.Update(id, req)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to update task", http.StatusInternalServerError)
		return
	}

	h.invalidateAfterChange(before.AssignedToUserID, task.AssignedToUserID)
	h.publishEvent(messaging.EventUpdated, task.ID, task.UserID, task.Status, before.AssignedToUserID, task.AssignedToUserID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	if _, ok := r.Context().Value(middleware.UserIDKey).(int64); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := parseTaskID(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}

	before, err := h.repo.GetByID(id)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get task", http.StatusInternalServerError)
		return
	}

	if err := h.repo.DeleteByID(id); err != nil {
		if err == repository.ErrTaskNotFound {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to delete task", http.StatusInternalServerError)
		return
	}

	h.invalidateAfterChange(before.AssignedToUserID)
	h.publishEvent(messaging.EventDeleted, id, before.UserID, before.Status, before.AssignedToUserID)

	w.WriteHeader(http.StatusNoContent)
}

func normalizeScope(scope string) string {
	switch scope {
	case "mine", "completed":
		return scope
	default:
		return "all"
	}
}

func (h *TaskHandler) invalidateAfterChange(assigneeIDs ...*int64) {
	var ids []int64
	for _, ptr := range assigneeIDs {
		if ptr != nil && *ptr > 0 {
			ids = append(ids, *ptr)
		}
	}
	h.cache.InvalidateLists(ids...)
}

func (h *TaskHandler) publishEvent(eventType string, taskID, userID int64, status string, assigneeIDs ...*int64) {
	if h.publisher == nil {
		return
	}
	var ids []int64
	seen := map[int64]bool{}
	for _, ptr := range assigneeIDs {
		if ptr != nil && *ptr > 0 && !seen[*ptr] {
			seen[*ptr] = true
			ids = append(ids, *ptr)
		}
	}
	h.publisher.Publish(messaging.TaskEvent{
		Type:              eventType,
		TaskID:            taskID,
		UserID:            userID,
		AssignedToUserIDs: ids,
		Status:            status,
	})
}

func parseTaskID(path string) (int64, error) {
	path = strings.TrimPrefix(path, "/tasks/")
	path = strings.TrimSuffix(path, "/take")
	path = strings.TrimSuffix(path, "/assign")
	path = strings.TrimSuffix(path, "/status")
	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil || id <= 0 {
		return 0, err
	}
	return id, nil
}
