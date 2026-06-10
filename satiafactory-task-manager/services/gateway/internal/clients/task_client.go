package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type TaskClient struct {
	baseURL string
	client  *http.Client
}

func NewTaskClient(baseURL string) *TaskClient {
	return &TaskClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

type Task struct {
	ID                  int64   `json:"id"`
	UserID              int64   `json:"user_id"`
	CreatorName         string  `json:"creator_name,omitempty"`
	AssignedToUserID    *int64  `json:"assigned_to_user_id,omitempty"`
	AssigneeName        string  `json:"assignee_name,omitempty"`
	Title               string  `json:"title"`
	Description         string  `json:"description,omitempty"`
	Status              string  `json:"status"`
	CreatedAt           string  `json:"created_at"`
	TargetItemClassName string  `json:"target_item_class_name,omitempty"`
	TargetAmount        float64 `json:"target_amount,omitempty"`
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

func (c *TaskClient) GetTasks(token string, scope string) ([]Task, error) {
	u := c.baseURL + "/tasks"
	if scope == "mine" || scope == "completed" {
		u += "?scope=" + scope
	}
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get tasks failed: %s", resp.Status)
	}
	var tasks []Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (c *TaskClient) GetTask(token string, id int64) (*Task, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/tasks/%d", c.baseURL, id), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get task failed: %s", resp.Status)
	}
	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (c *TaskClient) CreateTask(token string, req CreateTaskRequest) (*Task, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", c.baseURL+"/tasks", bytes.NewReader(body))
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create task failed: %s", resp.Status)
	}
	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (c *TaskClient) UpdateTask(token string, id int64, req UpdateTaskRequest) error {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("PATCH", fmt.Sprintf("%s/tasks/%d", c.baseURL, id), bytes.NewReader(body))
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update task failed: %s", resp.Status)
	}
	return nil
}

func (c *TaskClient) TakeTask(token string, id int64) error {
	httpReq, _ := http.NewRequest("POST", fmt.Sprintf("%s/tasks/%d/take", c.baseURL, id), nil)
	httpReq.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("take task failed: %s", resp.Status)
	}
	return nil
}

func (c *TaskClient) DeleteTask(token string, taskID int64) error {
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/tasks/%d", c.baseURL, taskID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete task failed: %s", resp.Status)
	}
	return nil
}
