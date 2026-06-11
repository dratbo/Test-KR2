package handlers

import (
	"html/template"
	"log"
	"net/http"
	"net/url"
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
	HubTier          int
	HubTierLabel     string
}

const tasksPerPage = 5

func NewTaskHandler(taskClient *clients.TaskClient, userClient *clients.UserClient, dataClient *clients.DataClient) (*TaskHandler, error) {
	funcMap := template.FuncMap{
		"formatItem":  formatItemName,
		"statusLabel": statusLabel,
		"add":         func(a, b int) int { return a + b },
		"sub":         func(a, b int) int { return a - b },
		"queryEscape": func(s string) string { return url.QueryEscape(s) },
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
		HubTier:          production.NormalizeHubTier(task.HubTier),
		HubTierLabel:     production.HubTierLabel(task.HubTier),
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
		view.Ingredients = append(view.Ingredients, ingredientRow{
			Name:      itemDisplayName(h.dataClient, ing.ItemClassName),
			Class:     ing.ItemClassName,
			Amount:    ingredientRatePerMin(recipe, view.TargetAmount, ing.Amount),
			IconURL:   clients.ItemIconURL(ing.ItemClassName),
			Craftable: ingredientCraftable(h.dataClient, ing.ItemClassName),
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
	return view
}

type taskProductionData struct {
	TaskID             int64
	HubTier            int
	HubTierLabel       string
	AvailableShards    int
	ConveyorMk         int
	PipeMk             int
	ConveyorLabel      string
	PipeLabel          string
	ShowFactoryDetails bool
	FactoryPlan        *production.FactoryPlan
	ProductionPlan     *production.StepPlan
}

func (h *TaskHandler) buildTaskProduction(task clients.Task, settings productionSettings) taskProductionData {
	hubTier := production.NormalizeHubTier(task.HubTier)
	data := taskProductionData{
		TaskID:             task.ID,
		HubTier:            hubTier,
		HubTierLabel:       production.HubTierLabel(hubTier),
		AvailableShards:    settings.Shards,
		ConveyorMk:         settings.ConveyorMk,
		PipeMk:             settings.PipeMk,
		ConveyorLabel:      production.ConveyorLabel(settings.ConveyorMk),
		PipeLabel:          production.PipeLabel(settings.PipeMk),
		ShowFactoryDetails: settings.ShowFactoryDetails,
	}
	if task.TargetItemClassName == "" || !settings.ShowFactoryDetails {
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
	data.FactoryPlan = buildFactoryPlan(h.dataClient, task.ID, task.TargetItemClassName, targetPerMin, settings.Shards, hubTier, settings.Logistics)
	rootBudget := rootShardBudget(h.dataClient, task.TargetItemClassName, targetPerMin, settings.Shards, hubTier)
	data.ProductionPlan = planForTaskRecipe(h.dataClient, recipe, targetPerMin, settings.Shards, rootBudget, hubTier, settings.Logistics)
	return data
}

func rootShardBudget(dataClient *clients.DataClient, recipeClass string, targetPerMin float64, shards, hubTier int) int {
	if shards <= 0 {
		return 0
	}
	plan := buildChainShardPlan(dataClient, recipeClass, targetPerMin, shards, hubTier)
	if plan == nil {
		return shards
	}
	return plan.rootBudget()
}

type tasksListData struct {
	Tasks      []TaskView
	Scope      string
	Query      string
	Page       int
	TotalPages int
	TotalCount int
	HasPrev    bool
	HasNext    bool
}

func taskMatchesQuery(v TaskView, q string) bool {
	if q == "" {
		return true
	}
	haystack := strings.ToLower(strings.Join([]string{
		v.Title,
		v.Description,
		v.RecipeName,
		v.CreatorName,
		v.AssigneeName,
		v.StatusLabel,
		statusLabel(v.Status),
	}, " "))
	return strings.Contains(haystack, q)
}

func filterTaskViews(views []TaskView, query string) []TaskView {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return views
	}
	out := make([]TaskView, 0, len(views))
	for _, v := range views {
		if taskMatchesQuery(v, q) {
			out = append(out, v)
		}
	}
	return out
}

func paginateTaskViews(views []TaskView, page int) ([]TaskView, int, int) {
	total := len(views)
	totalPages := 1
	if total > 0 {
		totalPages = (total + tasksPerPage - 1) / tasksPerPage
	}
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}
	start := (page - 1) * tasksPerPage
	if start >= total {
		return []TaskView{}, page, totalPages
	}
	end := start + tasksPerPage
	if end > total {
		end = total
	}
	return views[start:end], page, totalPages
}

func (h *TaskHandler) GetTasks(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	scope := r.URL.Query().Get("scope")
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	tasks, err := h.taskClient.GetTasks(cookie.Value, scope)
	if err != nil {
		http.Error(w, "Failed to load tasks", http.StatusInternalServerError)
		return
	}

	views := make([]TaskView, 0, len(tasks))
	for _, t := range tasks {
		views = append(views, h.enrichTask(t))
	}
	filtered := filterTaskViews(views, query)
	totalCount := len(filtered)
	paged, page, totalPages := paginateTaskViews(filtered, page)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := tasksListData{
		Tasks:      paged,
		Scope:      scope,
		Query:      query,
		Page:       page,
		TotalPages: totalPages,
		TotalCount: totalCount,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
	}
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
	productionData := h.buildTaskProduction(*task, productionSettingsFromTask(*task))

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
	prodSettings := parseProductionSettings(r)
	if r.URL.Query().Get("calc") == "1" || r.FormValue("calc") == "1" {
		shards := prodSettings.Shards
		conveyorMk := prodSettings.ConveyorMk
		pipeMk := prodSettings.PipeMk
		if err := h.taskClient.UpdateTask(cookie.Value, id, clients.UpdateTaskRequest{
			ProductionShards: &shards,
			ConveyorMk:       &conveyorMk,
			PipeMk:           &pipeMk,
		}); err != nil {
			log.Printf("save production settings for task %d: %v", id, err)
		}
	}
	data := h.buildTaskProduction(*task, prodSettings)

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

	hubTier, _ := strconv.Atoi(r.FormValue("hub_tier"))
	if hubTier <= 0 {
		hubTier = 9
	}

	if recipeClass != "" {
		unlockIndex, _ := h.dataClient.GetUnlockIndex()
		tierCtx := production.NewTierContext(hubTier, unlockIndex)
		if !tierCtx.RecipeUnlocked(recipeClass) {
			minTier := tierCtx.RecipeMinTier(recipeClass)
			msg := "Рецепт недоступен на выбранном тире HUB"
			if minTier > 0 {
				msg += ". Требуется " + production.HubTierLabel(minTier)
			}
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
	}

	_, err = h.taskClient.CreateTask(cookie.Value, clients.CreateTaskRequest{
		Title:               title,
		Description:         description,
		TargetItemClassName: recipeClass,
		TargetAmount:        targetAmount,
		HubTier:             hubTier,
		AssignedToUserID:    assignedTo,
	})
	if err != nil {
		log.Printf("CreateTask error: %v", err)
		http.Error(w, "Не удалось создать задачу. Сервис задач временно недоступен — обновите страницу и попробуйте снова.", http.StatusInternalServerError)
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

func (h *TaskHandler) UpdateHubTier(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/tasks/tier/")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	hubTier, _ := strconv.Atoi(r.FormValue("hub_tier"))
	if hubTier <= 0 {
		hubTier = 9
	}
	if err := h.taskClient.UpdateTask(cookie.Value, id, clients.UpdateTaskRequest{HubTier: &hubTier}); err != nil {
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
