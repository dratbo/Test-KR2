package handlers

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/dratbo/satisfactory-task-manager/gateway/internal/clients"
)

type TaskHandler struct {
	taskClient *clients.TaskClient
	dataClient *clients.DataClient
	tasksTmpl  *template.Template
}

func NewTaskHandler(taskClient *clients.TaskClient, dataClient *clients.DataClient) (*TaskHandler, error) {
	tasksTmpl, err := template.ParseFiles("templates/tasks.html")
	if err != nil {
		return nil, err
	}
	return &TaskHandler{
		taskClient: taskClient,
		dataClient: dataClient,
		tasksTmpl:  tasksTmpl,
	}, nil
}

// Index – главная страница (форма + список задач)
func (h *TaskHandler) Index(w http.ResponseWriter, r *http.Request) {
	// Проверяем авторизацию
	_, err := r.Cookie("token")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	// Получаем список предметов
	items, err := h.dataClient.GetItems()
	if err != nil {
		http.Error(w, "Failed to load items", http.StatusInternalServerError)
		return
	}
	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	tmpl.Execute(w, items)
}

// GetTasks – возвращает HTML-список задач (для htmx)
func (h *TaskHandler) GetTasks(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	tasks, err := h.taskClient.GetTasks(cookie.Value)
	if err != nil {
		http.Error(w, "Failed to load tasks", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = h.tasksTmpl.Execute(w, tasks)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

// CreateTask – создаёт задачу из POST-формы
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	description := strings.TrimSpace(r.FormValue("description"))
	targetItem := r.FormValue("target_item_class_name")
	targetAmount, _ := strconv.ParseFloat(r.FormValue("target_amount"), 64)

	if title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	// Логируем попытку создания
	log.Printf("Creating task: title=%s, targetItem=%s, amount=%f", title, targetItem, targetAmount)

	_, err = h.taskClient.CreateTask(cookie.Value, clients.CreateTaskRequest{
		Title:               title,
		Description:         description,
		TargetItemClassName: targetItem,
		TargetAmount:        targetAmount,
	})
	if err != nil {
		log.Printf("CreateTask error: %v", err)
		http.Error(w, "Failed to create task", http.StatusInternalServerError)
		return
	}

	log.Printf("Task created, returning tasks list")
	// Возвращаем обновлённый список задач
	h.GetTasks(w, r)
}

// DeleteTask – удаляет задачу
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/tasks/delete/")
	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "Invalid task id", http.StatusBadRequest)
		return
	}
	if err := h.taskClient.DeleteTask(cookie.Value, id); err != nil {
		http.Error(w, "Failed to delete task", http.StatusInternalServerError)
		return
	}
	h.GetTasks(w, r)
}
