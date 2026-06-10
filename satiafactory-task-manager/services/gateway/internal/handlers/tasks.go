package handlers

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/dratbo/satisfactory-task-manager/gateway/internal/clients"
	"github.com/dratbo/satisfactory-task-manager/gateway/internal/production"
)

type TaskHandler struct {
	taskClient     *clients.TaskClient
	userClient     *clients.UserClient
	dataClient     *clients.DataClient
	tasksTmpl      *template.Template
	detailTmpl     *template.Template
	productionTmpl *template.Template
}

type TaskView struct {
	ID           int64
	Title        string
	Description  string
	Status       string
	StatusLabel  string
	CreatedAt    string
	TargetAmount          float64
	TargetItemClassName   string
	RecipeName            string
	ProductName  string
	IconURL      string
	CreatorName      string
	AssigneeName     string
	AssignedToUserID *int64
	Ingredients    []ingredientRow
	Products       []ingredientRow
	Duration         float64
	ProductionPlan   *production.StepPlan
	RootTotalItems   float64
	RootRequiredRate float64
}

func NewTaskHandler(taskClient *clients.TaskClient, userClient *clients.UserClient, dataClient *clients.DataClient) (*TaskHandler, error) {
	funcMap := template.FuncMap{
		"formatItem":  formatItemName,
		"statusLabel": statusLabel,
	}
	tasksTmpl, err := template.New("tasks.html").Funcs(funcMap).ParseFiles("templates/tasks.html")
	if err != nil {
		return nil, err
	}
	detailTmpl, err := template.New("task_detail.html").Funcs(funcMap).ParseFiles(
		"templates/task_detail.html",
		"templates/task_production.html",
		"templates/factory_plan.html",
		"templates/production_plan.html",
	)
	if err != nil {
		return nil, err
	}
	productionTmpl, err := template.New("task_production.html").Funcs(funcMap).ParseFiles(
		"templates/task_production.html",
		"templates/factory_plan.html",
		"templates/production_plan.html",
	)
	if err != nil {
		return nil, err
	}
	return &TaskHandler{
		taskClient:     taskClient,
		userClient:     userClient,
		dataClient:     dataClient,
		tasksTmpl:      tasksTmpl,
		detailTmpl:     detailTmpl,
		productionTmpl: productionTmpl,
	}, nil
}

func statusLabel(s string) string {
	switch s {
	case "in_progress":
		return "В работе"
	case "completed":
		return "Выполнено"
	default:
		return "Ожидает"
	}
}

func formatItemName(className string) string {
	s := strings.TrimSuffix(className, "_C")
	s = strings.TrimPrefix(s, "Desc_")
	s = strings.ReplaceAll(s, "_", " ")
	return s
}

type pageNav struct {
	Nav string
}

func (h *TaskHandler) Index(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie("token"); err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	tmpl := template.Must(template.ParseFiles(
		"templates/site_header.html",
		"templates/scripts_tasks.html",
		"templates/index.html",
	))
	if err := tmpl.ExecuteTemplate(w, "index.html", pageNav{Nav: "home"}); err != nil {
		log.Printf("render index: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *TaskHandler) MyTasks(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie("token"); err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	tmpl := template.Must(template.ParseFiles(
		"templates/site_header.html",
		"templates/scripts_tasks.html",
		"templates/my_tasks.html",
	))
	if err := tmpl.ExecuteTemplate(w, "my_tasks.html", pageNav{Nav: "my-tasks"}); err != nil {
		log.Printf("render my_tasks: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *TaskHandler) enrichTask(task clients.Task) TaskView {
	view := TaskView{
		ID:           task.ID,
		Title:        task.Title,
		Description:  task.Description,
		Status:       task.Status,
		StatusLabel:  statusLabel(task.Status),
		CreatedAt:    task.CreatedAt,
		TargetAmount: task.TargetAmount,
		CreatorName:      task.CreatorName,
		AssigneeName:     task.AssigneeName,
		AssignedToUserID: task.AssignedToUserID,
	}
	if task.TargetAmount <= 0 {
		view.TargetAmount = 1
	}
	view.TargetItemClassName = task.TargetItemClassName
	if task.TargetItemClassName == "" {
		return view
	}

	recipe, err := h.dataClient.GetRecipe(task.TargetItemClassName)
	if err != nil || recipe == nil {
		view.RecipeName = formatItemName(task.TargetItemClassName)
		return view
	}

	view.RecipeName = recipeDisplayTitle(recipe)
	view.Duration = recipe.Duration
	for _, ing := range recipe.Ingredients {
		craftable, _ := h.dataClient.HasRecipeForProduct(ing.ItemClassName)
		view.Ingredients = append(view.Ingredients, ingredientRow{
			Name:      itemDisplayName(h.dataClient, ing.ItemClassName),
			Class:     ing.ItemClassName,
			Amount:    ingredientRatePerMin(recipe, view.TargetAmount, ing.Amount),
			IconURL:   clients.ItemIconURL(ing.ItemClassName),
			Craftable: craftable,
		})
	}
	for _, prod := range recipe.Products {
		view.Products = append(view.Products, ingredientRow{
			Name:    itemDisplayName(h.dataClient, prod.ItemClassName),
			Class:   prod.ItemClassName,
			Amount:  productRatePerMin(recipe, view.TargetAmount, prod.Amount),
			IconURL: clients.ItemIconURL(prod.ItemClassName),
		})
		if view.IconURL == "" || view.IconURL == "/static/placeholder.svg" {
			view.IconURL = clients.ItemIconURL(prod.ItemClassName)
		}
	}
	params := production.RootPlanParamsFromTask(view.TargetAmount, 0)
	if len(recipe.Products) > 0 {
		params = production.RootPlanParamsFromTask(view.TargetAmount, recipe.Products[0].Amount)
	}
	view.RootTotalItems = params.RequiredRate
	view.RootRequiredRate = params.RequiredRate
	view.ProductionPlan = planForTaskRecipe(h.dataClient, recipe, view.TargetAmount, 0, 0)
	return view
}

type taskProductionData struct {
	TaskID          int64
	AvailableShards int
	FactoryPlan     *production.FactoryPlan
	ProductionPlan  *production.StepPlan
}

func (h *TaskHandler) buildTaskProduction(task clients.Task, shards int) taskProductionData {
	data := taskProductionData{
		TaskID:          task.ID,
		AvailableShards: shards,
	}
	if task.TargetItemClassName == "" {
		return data
	}
	recipe, err := h.dataClient.GetRecipe(task.TargetItemClassName)
	if err != nil || recipe == nil {
		return data
	}
	targetPerMin := task.TargetAmount
	if targetPerMin <= 0 {
		targetPerMin = 1
	}
	data.FactoryPlan = buildFactoryPlan(h.dataClient, task.ID, task.TargetItemClassName, targetPerMin, shards)
	rootBudget := rootShardBudget(h.dataClient, task.TargetItemClassName, targetPerMin, shards)
	data.ProductionPlan = planForTaskRecipe(h.dataClient, recipe, targetPerMin, shards, rootBudget)
	return data
}

func rootShardBudget(dataClient *clients.DataClient, recipeClass string, targetPerMin float64, shards int) int {
	if shards <= 0 {
		return 0
	}
	recipe, err := dataClient.GetRecipe(recipeClass)
	if err != nil || recipe == nil || len(recipe.Products) == 0 {
		return shards
	}
	params := production.RootPlanParamsFromTask(targetPerMin, recipe.Products[0].Amount)
	steps := collectChainSteps(dataClient, recipe.Products[0].ItemClassName, params.RequiredRate, 0)
	if len(steps) == 0 {
		return shards
	}
	overclockable := make([]bool, len(steps))
	for i, s := range steps {
		if s.raw {
			overclockable[i] = production.SupportsPowerShards(production.DefaultMinerClass)
		} else if s.recipe != nil {
			overclockable[i] = production.SupportsPowerShards(production.PickFactoryBuilding(s.recipe.ProducedIn))
		}
	}
	budgets := production.DistributeShardBudget(len(steps), overclockable, shards)
	return budgets[len(budgets)-1]
}

func (h *TaskHandler) GetTasks(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	scope := r.URL.Query().Get("scope")
	tasks, err := h.taskClient.GetTasks(cookie.Value, scope)
	if err != nil {
		http.Error(w, "Failed to load tasks", http.StatusInternalServerError)
		return
	}

	views := make([]TaskView, 0, len(tasks))
	for _, t := range tasks {
		views = append(views, h.enrichTask(t))
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct {
		Tasks []TaskView
		Scope string
	}{Tasks: views, Scope: scope}
	if err := h.tasksTmpl.Execute(w, data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *TaskHandler) GetTaskDetail(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/tasks/detail/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	task, err := h.taskClient.GetTask(cookie.Value, id)
	if err != nil || task == nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	users, _ := h.userClient.ListUsers()
	view := h.enrichTask(*task)
	shards := parseShardCount(r)
	productionData := h.buildTaskProduction(*task, shards)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.detailTmpl.Execute(w, struct {
		Task           TaskView
		Users          []clients.UserBrief
		FactoryPlan    *production.FactoryPlan
		ProductionData taskProductionData
	}{
		Task:           view,
		Users:          users,
		FactoryPlan:    productionData.FactoryPlan,
		ProductionData: productionData,
	}); err != nil {
		log.Printf("task detail template: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *TaskHandler) GetTaskProduction(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie("token"); err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/tasks/production/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	cookie, _ := r.Cookie("token")
	task, err := h.taskClient.GetTask(cookie.Value, id)
	if err != nil || task == nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	shards := parseShardCount(r)
	data := h.buildTaskProduction(*task, shards)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.productionTmpl.Execute(w, data)
}

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
	recipeClass := strings.TrimSpace(r.FormValue("target_item_class_name"))
	targetAmount, _ := strconv.ParseFloat(r.FormValue("target_amount"), 64)
	if targetAmount <= 0 {
		targetAmount = 1
	}

	var assignedTo *int64
	if v := strings.TrimSpace(r.FormValue("assigned_to_user_id")); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil && id > 0 {
			assignedTo = &id
		}
	}

	if title == "" && recipeClass != "" {
		if recipe, err := h.dataClient.GetRecipe(recipeClass); err == nil && recipe != nil {
			title = recipe.DisplayName
		}
	}
	if title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	log.Printf("Creating task: title=%s, recipe=%s, amount=%f", title, recipeClass, targetAmount)

	_, err = h.taskClient.CreateTask(cookie.Value, clients.CreateTaskRequest{
		Title:               title,
		Description:         description,
		TargetItemClassName: recipeClass,
		TargetAmount:        targetAmount,
		AssignedToUserID:    assignedTo,
	})
	if err != nil {
		log.Printf("CreateTask error: %v", err)
		http.Error(w, "Failed to create task", http.StatusInternalServerError)
		return
	}

	scope := r.FormValue("task_scope")
	if scope == "" {
		scope = r.URL.Query().Get("scope")
	}
	r2 := r.Clone(r.Context())
	q := r2.URL.Query()
	q.Set("scope", scope)
	r2.URL.RawQuery = q.Encode()
	h.GetTasks(w, r2)
}

func (h *TaskHandler) TakeTask(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/tasks/take/")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if err := h.taskClient.TakeTask(cookie.Value, id); err != nil {
		http.Error(w, "Failed", http.StatusInternalServerError)
		return
	}
	r2 := r.Clone(r.Context())
	r2.URL.Path = "/tasks/detail/" + idStr
	h.GetTaskDetail(w, r2)
}

func (h *TaskHandler) UpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/tasks/status/")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	status := r.FormValue("status")
	if err := h.taskClient.UpdateTask(cookie.Value, id, clients.UpdateTaskRequest{Status: &status}); err != nil {
		http.Error(w, "Failed", http.StatusInternalServerError)
		return
	}
	r2 := r.Clone(r.Context())
	r2.URL.Path = "/tasks/detail/" + idStr
	h.GetTaskDetail(w, r2)
}

func (h *TaskHandler) AssignTask(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/tasks/assign/")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	uid, _ := strconv.ParseInt(r.FormValue("assigned_to_user_id"), 10, 64)
	status := "in_progress"
	req := clients.UpdateTaskRequest{AssignedToUserID: &uid, Status: &status}
	if uid == 0 {
		req.AssignedToUserID = nil
		pending := "pending"
		req.Status = &pending
	}
	if err := h.taskClient.UpdateTask(cookie.Value, id, req); err != nil {
		http.Error(w, "Failed", http.StatusInternalServerError)
		return
	}
	r2 := r.Clone(r.Context())
	r2.URL.Path = "/tasks/detail/" + idStr
	h.GetTaskDetail(w, r2)
}

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
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = r.FormValue("scope")
	}
	r2 := r.Clone(r.Context())
	q := r2.URL.Query()
	q.Set("scope", scope)
	r2.URL.RawQuery = q.Encode()
	h.GetTasks(w, r2)
}
