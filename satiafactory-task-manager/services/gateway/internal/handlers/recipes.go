package handlers

import (
	"html/template"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/dratbo/satisfactory-task-manager/gateway/internal/clients"
	"github.com/dratbo/satisfactory-task-manager/gateway/internal/production"
)

type RecipeHandler struct {
	dataClient    *clients.DataClient
	searchTmpl    *template.Template
	previewTmpl   *template.Template
	chainTmpl     *template.Template
	chainRevTmpl  *template.Template
}

func NewRecipeHandler(dataClient *clients.DataClient) (*RecipeHandler, error) {
	funcMap := template.FuncMap{
		"formatItem": formatItemName,
		"add":        func(a, b int) int { return a + b },
	}
	searchTmpl, err := template.ParseFiles("templates/recipes_search.html")
	if err != nil {
		return nil, err
	}
	previewTmpl, err := template.New("recipe_preview.html").Funcs(funcMap).ParseFiles("templates/recipe_preview.html")
	if err != nil {
		return nil, err
	}
	chainTmpl, err := template.New("recipe_chain.html").Funcs(funcMap).ParseFiles(
		"templates/recipe_chain.html",
		"templates/production_plan.html",
	)
	if err != nil {
		return nil, err
	}
	chainRevTmpl, err := template.New("recipe_chain_reverse.html").Funcs(funcMap).ParseFiles("templates/recipe_chain_reverse.html")
	if err != nil {
		return nil, err
	}
	return &RecipeHandler{
		dataClient:   dataClient,
		searchTmpl:   searchTmpl,
		previewTmpl:  previewTmpl,
		chainTmpl:    chainTmpl,
		chainRevTmpl: chainRevTmpl,
	}, nil
}

type recipeSearchRow struct {
	ClassName     string
	DisplayName   string
	EnglishName   string
	DisplayNameRU string
	ProductName   string
	IconURL       string
	IsAlternate   bool
}

type recipeSearchData struct {
	Rows      []recipeSearchRow
	EmptyHint string
}

func (h *RecipeHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if len(q) < 2 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<p class="hint">Введите минимум 2 символа для поиска рецепта…</p>`))
		return
	}

	hubTier, _ := strconv.Atoi(r.URL.Query().Get("hub_tier"))
	if hubTier <= 0 {
		hubTier = 9
	}
	unlockIndex, _ := h.dataClient.GetUnlockIndex()
	tierCtx := production.NewTierContext(hubTier, unlockIndex)

	includeAlternates := r.URL.Query().Get("include_alternates") == "1" || r.URL.Query().Get("include_alternates") == "on"
	recipes, err := h.dataClient.SearchRecipes(q, includeAlternates)
	if err != nil {
		http.Error(w, "Не удалось загрузить рецепты", http.StatusBadGateway)
		return
	}

	rows := make([]recipeSearchRow, 0, len(recipes))
	for _, rec := range recipes {
		if !tierCtx.RecipeUnlocked(rec.ClassName) {
			continue
		}
		title := rec.DisplayName
		englishName := ""
		if rec.DisplayNameRU != "" {
			title = rec.DisplayNameRU
			englishName = rec.DisplayName
		}
		row := recipeSearchRow{
			ClassName:     rec.ClassName,
			DisplayName:   title,
			EnglishName:   englishName,
			DisplayNameRU: rec.DisplayNameRU,
			IsAlternate:   strings.Contains(rec.ClassName, "Alternate") || strings.HasPrefix(rec.DisplayName, "Alternate:"),
		}
		if len(rec.Products) > 0 {
			if production.IsExtractedResource(rec.Products[0].ItemClassName) {
				continue
			}
			row.ProductName = rec.Products[0].ItemClassName
			row.IconURL = clients.ItemIconURL(rec.Products[0].ItemClassName)
		}
		rows = append(rows, row)
	}

	data := recipeSearchData{Rows: rows}
	if len(rows) == 0 {
		data.EmptyHint = "Для " + production.HubTierLabel(hubTier) + " нет доступных рецептов по запросу «" + q + "». Выберите более высокий тир HUB."
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.searchTmpl.Execute(w, data)
}

type ingredientRow struct {
	Name      string
	Class     string
	Amount    float64
	IconURL   string
	Craftable bool
}

type recipeChainOption struct {
	ClassName   string
	DisplayName string
	IsAlternate bool
}

type recipeChainData struct {
	ItemClass         string
	ItemName          string
	IconURL           string
	Amount            float64
	Depth             int
	IncludeAlternates bool
	SelectedRecipe    string
	RecipeOptions     []recipeChainOption
	RecipeClass       string
	RecipeTitle       string
	IsAlternate       bool
	Duration          float64
	Ingredients       []ingredientRow
	Products          []ingredientRow
	RawResource       bool
	ProductionPlan    *production.StepPlan
	RootTotalItems    float64
	RootRequiredRate  float64
	RootRecipeClass   string
	AvailableShards   int
	HubTier           int
}

type recipePreviewData struct {
	RecipeClass string
	Title       string
	IconURL     string
	Multiplier  float64
	Ingredients []ingredientRow
	Products    []ingredientRow
	Duration    float64
}

func (h *RecipeHandler) Preview(w http.ResponseWriter, r *http.Request) {
	className := strings.TrimSpace(r.URL.Query().Get("recipe"))
	if className == "" {
		className = strings.TrimSpace(r.URL.Query().Get("target_item_class_name"))
	}
	if className == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(""))
		return
	}

	amountStr := r.URL.Query().Get("amount")
	if amountStr == "" {
		amountStr = r.URL.Query().Get("target_amount")
	}
	targetPerMin, _ := strconv.ParseFloat(amountStr, 64)
	if targetPerMin <= 0 {
		targetPerMin = 1
	}

	recipe, err := h.dataClient.GetRecipe(className)
	if err != nil || recipe == nil {
		http.Error(w, "Рецепт не найден", http.StatusNotFound)
		return
	}

	data := recipePreviewData{
		RecipeClass: recipe.ClassName,
		Title:       recipeDisplayTitle(recipe),
		Multiplier:  targetPerMin,
		Duration:    recipe.Duration,
	}

	for _, ing := range recipe.Ingredients {
		data.Ingredients = append(data.Ingredients, ingredientRow{
			Name:    itemDisplayName(h.dataClient, ing.ItemClassName),
			Class:   ing.ItemClassName,
			Amount:  ingredientRatePerMin(recipe, targetPerMin, ing.Amount),
			IconURL: clients.ItemIconURL(ing.ItemClassName),
		})
	}
	for _, prod := range recipe.Products {
		data.Products = append(data.Products, ingredientRow{
			Name:    itemDisplayName(h.dataClient, prod.ItemClassName),
			Class:   prod.ItemClassName,
			Amount:  productRatePerMin(recipe, targetPerMin, prod.Amount),
			IconURL: clients.ItemIconURL(prod.ItemClassName),
		})
		if data.IconURL == "" {
			data.IconURL = clients.ItemIconURL(prod.ItemClassName)
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.previewTmpl.Execute(w, data)
}

func (h *RecipeHandler) Chain(w http.ResponseWriter, r *http.Request) {
	itemClass := strings.TrimSpace(r.URL.Query().Get("item"))
	if itemClass == "" {
		http.Error(w, "item required", http.StatusBadRequest)
		return
	}

	amount, _ := strconv.ParseFloat(r.URL.Query().Get("amount"), 64)
	if amount <= 0 {
		amount = 1
	}
	depth, _ := strconv.Atoi(r.URL.Query().Get("depth"))
	if depth <= 0 {
		depth = 1
	}
	includeAlternates := r.URL.Query().Get("include_alternates") == "1" || r.URL.Query().Get("include_alternates") == "on"
	selectedRecipe := strings.TrimSpace(r.URL.Query().Get("recipe"))
	rootTotal, _ := strconv.ParseFloat(r.URL.Query().Get("root_total"), 64)
	rootRate, _ := strconv.ParseFloat(r.URL.Query().Get("root_rate"), 64)
	shards := parseShardCount(r)
	hubTier, _ := strconv.Atoi(firstNonEmpty(r.URL.Query().Get("hub_tier"), r.FormValue("hub_tier")))
	if hubTier <= 0 {
		hubTier = 9
	}
	rootRecipe := strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("root_recipe"), r.FormValue("root_recipe")))

	itemName := itemDisplayName(h.dataClient, itemClass)

	data := recipeChainData{
		ItemClass:         itemClass,
		ItemName:          itemName,
		IconURL:           clients.ItemIconURL(itemClass),
		Amount:            amount,
		Depth:             depth,
		IncludeAlternates: includeAlternates,
		SelectedRecipe:    selectedRecipe,
		RootTotalItems:    rootTotal,
		RootRequiredRate:  rootRate,
		RootRecipeClass:   rootRecipe,
		AvailableShards:   shards,
		HubTier:           hubTier,
	}

	if production.IsExtractedResource(itemClass) {
		h.renderExtractedChain(w, &data, itemClass, itemName, amount, rootRecipe, rootRate, shards, hubTier)
		return
	}

	recipes, err := h.dataClient.GetRecipesByProduct(itemClass, includeAlternates)
	if err != nil || len(recipes) == 0 {
		h.renderExtractedChain(w, &data, itemClass, itemName, amount, rootRecipe, rootRate, shards, hubTier)
		return
	}

	for _, rec := range recipes {
		title := rec.DisplayName
		if rec.DisplayNameRU != "" {
			title = rec.DisplayNameRU
		}
		data.RecipeOptions = append(data.RecipeOptions, recipeChainOption{
			ClassName:   rec.ClassName,
			DisplayName: title,
			IsAlternate: strings.Contains(rec.ClassName, "Alternate") || strings.HasPrefix(rec.DisplayName, "Alternate:"),
		})
	}

	var recipe *clients.Recipe
	for i := range recipes {
		if selectedRecipe != "" && recipes[i].ClassName == selectedRecipe {
			recipe = &recipes[i]
			break
		}
	}
	if recipe == nil {
		recipe = &recipes[0]
	}

	data.RecipeClass = recipe.ClassName
	data.RecipeTitle = recipeDisplayTitle(recipe)
	data.IsAlternate = strings.Contains(recipe.ClassName, "Alternate") || strings.HasPrefix(recipe.DisplayName, "Alternate:")
	data.Duration = recipe.Duration
	data.SelectedRecipe = recipe.ClassName

	for _, ing := range recipe.Ingredients {
		data.Ingredients = append(data.Ingredients, ingredientRow{
			Name:      itemDisplayName(h.dataClient, ing.ItemClassName),
			Class:     ing.ItemClassName,
			Amount:    ingredientRatePerMin(recipe, amount, ing.Amount),
			IconURL:   clients.ItemIconURL(ing.ItemClassName),
			Craftable: ingredientCraftable(h.dataClient, ing.ItemClassName),
		})
	}
	for _, prod := range recipe.Products {
		data.Products = append(data.Products, ingredientRow{
			Name:    itemDisplayName(h.dataClient, prod.ItemClassName),
			Class:   prod.ItemClassName,
			Amount:  productRatePerMin(recipe, amount, prod.Amount),
			IconURL: clients.ItemIconURL(prod.ItemClassName),
		})
	}

	shardBudget := shardBudgetForChainItem(h.dataClient, rootRecipe, rootRate, shards, hubTier, itemClass, recipe.ClassName, false)
	ctx := loadProductionContext(h.dataClient, data.RootTotalItems, data.RootRequiredRate, shards, hubTier, production.LogisticsParams{})
	data.ProductionPlan = ctx.planForRecipe(itemClass, itemName, amount, recipe, false, shardBudget)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.chainTmpl.Execute(w, data)
}

type chainTreeNode struct {
	ItemClass   string
	Amount      float64
	Name        string
	IconURL     string
	RecipeTitle string
	RawResource bool
	Tier        int
	Children    []*chainTreeNode
}

type reverseChainItem struct {
	ItemClass   string
	Amount      float64
	Name        string
	IconURL     string
	ChainKey    string
	RecipeTitle string
	RawResource bool
}

type reverseChainTier struct {
	Tier  int
	Label string
	First bool
	Items []reverseChainItem
}

type reverseChainData struct {
	Tiers []reverseChainTier
}

const maxChainTreeDepth = 14

func (h *RecipeHandler) renderExtractedChain(w http.ResponseWriter, data *recipeChainData, itemClass, itemName string, amount float64, rootRecipe string, rootRate float64, shards, hubTier int) {
	data.RawResource = true
	shardBudget := shardBudgetForChainItem(h.dataClient, rootRecipe, rootRate, shards, hubTier, itemClass, "", true)
	ctx := loadProductionContext(h.dataClient, data.RootTotalItems, data.RootRequiredRate, shards, hubTier, production.LogisticsParams{})
	data.ProductionPlan = ctx.planForRecipe(itemClass, itemName, amount, nil, true, shardBudget)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.chainTmpl.Execute(w, data)
}

func (h *RecipeHandler) buildChainTree(cache *recipeLookupCache, tier *production.TierContext, itemClass string, amount float64, depth int, preferredRecipe *clients.Recipe) *chainTreeNode {
	node := &chainTreeNode{
		ItemClass: itemClass,
		Amount:    amount,
		Name:      itemDisplayName(h.dataClient, itemClass),
		IconURL:   clients.ItemIconURL(itemClass),
	}
	if depth >= maxChainTreeDepth {
		node.RawResource = true
		return node
	}
	if production.IsExtractedResource(itemClass) {
		node.RawResource = true
		return node
	}

	var recipe *clients.Recipe
	if preferredRecipe != nil {
		recipe = preferredRecipe
	} else {
		recipes, err := cache.recipesByProduct(itemClass)
		if err != nil || len(recipes) == 0 {
			node.RawResource = true
			return node
		}
		recipe = production.PickChainRecipeWithTier(recipes, tier)
	}
	node.RecipeTitle = recipeDisplayTitle(recipe)
	prodPerCycle := recipeProductPerCycle(recipe)
	for _, ing := range recipe.Ingredients {
		childRate := round2(amount * ing.Amount / prodPerCycle)
		child := h.buildChainTree(cache, tier, ing.ItemClassName, childRate, depth+1, nil)
		node.Children = append(node.Children, child)
	}
	return node
}

func assignChainTiers(node *chainTreeNode) int {
	if len(node.Children) == 0 {
		node.Tier = 0
		return 0
	}
	maxChild := 0
	for _, child := range node.Children {
		if t := assignChainTiers(child); t > maxChild {
			maxChild = t
		}
	}
	node.Tier = maxChild + 1
	return node.Tier
}

func aggregateChainTiers(node *chainTreeNode, tiers map[int]map[string]*reverseChainItem) {
	if node.ItemClass != "" {
		if tiers[node.Tier] == nil {
			tiers[node.Tier] = map[string]*reverseChainItem{}
		}
		bucket := tiers[node.Tier]
		if existing, ok := bucket[node.ItemClass]; ok {
			existing.Amount = round2(existing.Amount + node.Amount)
			existing.ChainKey = node.ItemClass + ":" + formatChainAmount(existing.Amount)
		} else {
			bucket[node.ItemClass] = &reverseChainItem{
				ItemClass:   node.ItemClass,
				Amount:      node.Amount,
				Name:        node.Name,
				IconURL:     node.IconURL,
				ChainKey:    node.ItemClass + ":" + formatChainAmount(node.Amount),
				RecipeTitle: node.RecipeTitle,
				RawResource: node.RawResource,
			}
		}
	}
	for _, child := range node.Children {
		aggregateChainTiers(child, tiers)
	}
}

func formatChainAmount(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}

func tierLabel(tier, maxTier int) string {
	if tier == 0 {
		return "Сырьё и базовые ресурсы"
	}
	if tier == maxTier {
		return "Основная задача"
	}
	return "Промежуточный шаг " + strconv.Itoa(tier)
}

func (h *RecipeHandler) ChainReverse(w http.ResponseWriter, r *http.Request) {
	recipeClass := strings.TrimSpace(r.URL.Query().Get("recipe"))
	amount, _ := strconv.ParseFloat(r.URL.Query().Get("amount"), 64)
	if amount <= 0 {
		amount = 1
	}
	if recipeClass == "" {
		http.Error(w, "recipe required", http.StatusBadRequest)
		return
	}

	recipe, err := h.dataClient.GetRecipe(recipeClass)
	if err != nil || recipe == nil || len(recipe.Products) == 0 {
		http.Error(w, "Рецепт не найден", http.StatusNotFound)
		return
	}

	prod := recipe.Products[0]
	hubTier, _ := strconv.Atoi(r.URL.Query().Get("hub_tier"))
	if hubTier <= 0 {
		hubTier = 9
	}
	unlockIndex, _ := h.dataClient.GetUnlockIndex()
	tierCtx := production.NewTierContext(hubTier, unlockIndex)
	cache := newRecipeLookupCache(h.dataClient)
	root := h.buildChainTree(cache, tierCtx, prod.ItemClassName, productRatePerMin(recipe, amount, prod.Amount), 0, recipe)
	maxTier := assignChainTiers(root)

	tierMap := map[int]map[string]*reverseChainItem{}
	aggregateChainTiers(root, tierMap)

	data := reverseChainData{}
	for tier := 0; tier <= maxTier; tier++ {
		bucket := tierMap[tier]
		if len(bucket) == 0 {
			continue
		}
		rt := reverseChainTier{
			Tier:  tier,
			Label: tierLabel(tier, maxTier),
			First: tier == 0,
		}
		for _, item := range bucket {
			rt.Items = append(rt.Items, *item)
		}
		sort.Slice(rt.Items, func(i, j int) bool {
			return rt.Items[i].Name < rt.Items[j].Name
		})
		data.Tiers = append(data.Tiers, rt)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.chainRevTmpl.Execute(w, data)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
