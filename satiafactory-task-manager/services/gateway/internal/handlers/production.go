package handlers

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/dratbo/satisfactory-task-manager/gateway/internal/clients"
	"github.com/dratbo/satisfactory-task-manager/gateway/internal/production"
)

type productionContext struct {
	lookup          func(string) string
	rootTotalItems  float64
	rootRate        float64
	availableShards int
	tier            *production.TierContext
	logistics       production.LogisticsParams
}

func loadProductionContext(dataClient *clients.DataClient, rootTotalItems, rootRate float64, availableShards, hubTier int, logistics production.LogisticsParams) productionContext {
	buildings, _ := dataClient.GetBuildings()
	unlockIndex, _ := dataClient.GetUnlockIndex()
	return productionContext{
		lookup:          production.BuildingLookupFromList(buildings),
		rootTotalItems:  rootTotalItems,
		rootRate:        rootRate,
		availableShards: availableShards,
		tier:            production.NewTierContext(hubTier, unlockIndex),
		logistics:       logistics,
	}
}

func (ctx productionContext) planForRecipe(itemClass, itemName string, amount float64, recipe *clients.Recipe, raw bool, shardBudget int) *production.StepPlan {
	// amount is already a per-minute throughput for factory/chain steps.
	return production.BuildStepPlan(production.PlanInput{
		ItemClass:       itemClass,
		ItemName:        itemName,
		TotalItems:      amount,
		RequiredRate:    amount,
		Recipe:          recipe,
		RawResource:     raw,
		BuildingLookup:  ctx.lookup,
		ShardBudget:     shardBudget,
		AvailableShards: ctx.availableShards,
		Tier:            ctx.tier,
		Logistics:       ctx.logistics,
	})
}

func planForTaskRecipe(dataClient *clients.DataClient, recipe *clients.Recipe, targetPerMin float64, availableShards, shardBudget, hubTier int, logistics production.LogisticsParams) *production.StepPlan {
	if recipe == nil || len(recipe.Products) == 0 {
		return nil
	}
	productAmount := recipe.Products[0].Amount
	params := production.RootPlanParamsFromTask(targetPerMin, productAmount)

	prodClass := recipe.Products[0].ItemClassName
	itemName := itemDisplayName(dataClient, prodClass)

	ctx := loadProductionContext(dataClient, params.RequiredRate, params.RequiredRate, availableShards, hubTier, logistics)
	return ctx.planForRecipe(prodClass, itemName, params.RequiredRate, recipe, false, shardBudget)
}

type chainStep struct {
	itemClass string
	itemName  string
	amount    float64
	recipe    *clients.Recipe
	raw       bool
	skipPlan  bool
}

type recipeLookupCache struct {
	client    *clients.DataClient
	byProduct map[string][]clients.Recipe
}

func newRecipeLookupCache(client *clients.DataClient) *recipeLookupCache {
	return &recipeLookupCache{
		client:    client,
		byProduct: make(map[string][]clients.Recipe),
	}
}

func (c *recipeLookupCache) recipesByProduct(itemClass string) ([]clients.Recipe, error) {
	if recipes, ok := c.byProduct[itemClass]; ok {
		return recipes, nil
	}
	recipes, err := c.client.GetRecipesByProduct(itemClass, false)
	if err != nil {
		return nil, err
	}
	c.byProduct[itemClass] = recipes
	return recipes, nil
}

func itemDisplayNameFor(dataClient *clients.DataClient, itemClass string) string {
	return itemDisplayName(dataClient, itemClass)
}

func collectChainSteps(cache *recipeLookupCache, tier *production.TierContext, itemClass string, amount float64, depth int, preferredRecipe *clients.Recipe) []chainStep {
	name := itemDisplayNameFor(cache.client, itemClass)
	if depth >= maxChainTreeDepth {
		return []chainStep{{itemClass: itemClass, itemName: name, amount: amount, raw: true}}
	}
	if production.IsExtractedResource(itemClass) {
		return []chainStep{{itemClass: itemClass, itemName: name, amount: amount, raw: true}}
	}

	var recipe *clients.Recipe
	if preferredRecipe != nil {
		recipe = preferredRecipe
	} else {
		recipes, err := cache.recipesByProduct(itemClass)
		if err != nil || len(recipes) == 0 {
			step := chainStep{itemClass: itemClass, itemName: name, amount: amount, raw: true}
			if production.IsUnplannedRawItem(itemClass, false) {
				step.skipPlan = true
			}
			return []chainStep{step}
		}
		recipe = production.PickChainRecipeWithTier(recipes, tier)
	}
	prodAmount := 1.0
	if len(recipe.Products) > 0 {
		prodAmount = recipe.Products[0].Amount
	}

	var steps []chainStep
	for _, ing := range recipe.Ingredients {
		ingAmount := round2(ing.Amount * amount / prodAmount)
		steps = append(steps, collectChainSteps(cache, tier, ing.ItemClassName, ingAmount, depth+1, nil)...)
	}
	steps = append(steps, chainStep{
		itemClass: itemClass,
		itemName:  name,
		amount:    amount,
		recipe:    recipe,
		raw:       false,
	})
	return steps
}

func aggregateChainSteps(steps []chainStep) []chainStep {
	type key struct {
		itemClass   string
		recipeClass string
		raw         bool
		skipPlan    bool
	}
	merged := map[key]*chainStep{}
	order := make([]key, 0, len(steps))

	for _, s := range steps {
		recipeClass := ""
		if s.recipe != nil {
			recipeClass = s.recipe.ClassName
		}
		k := key{itemClass: s.itemClass, recipeClass: recipeClass, raw: s.raw, skipPlan: s.skipPlan}
		if existing, ok := merged[k]; ok {
			existing.amount = round2(existing.amount + s.amount)
			continue
		}
		copy := s
		merged[k] = &copy
		order = append(order, k)
	}

	out := make([]chainStep, 0, len(order))
	for _, k := range order {
		out = append(out, *merged[k])
	}
	return out
}

type chainShardPlan struct {
	steps   []chainStep
	budgets []int
}

func chainStepShardInput(s chainStep, tierCtx *production.TierContext) production.ShardStepInput {
	if s.raw {
		ext := production.PickExtractor(s.itemClass, tierCtx)
		return production.ShardStepInput{
			RequiredRate:  s.amount,
			BaseRate:      ext.BaseRate,
			Overclockable: production.SupportsPowerShards(ext.BuildingClass),
		}
	}
	if s.recipe == nil || len(s.recipe.Products) == 0 {
		return production.ShardStepInput{RequiredRate: s.amount}
	}
	buildingClass := production.PickFactoryBuilding(s.recipe.ProducedIn)
	return production.ShardStepInput{
		RequiredRate:  s.amount,
		BaseRate:      production.ItemsPerMinute(s.recipe.Products[0].Amount, s.recipe.Duration, 100),
		Overclockable: production.SupportsPowerShards(buildingClass),
	}
}

func buildChainShardPlan(dataClient *clients.DataClient, recipeClass string, targetPerMin float64, shards, hubTier int) *chainShardPlan {
	recipe, err := dataClient.GetRecipe(recipeClass)
	if err != nil || recipe == nil || len(recipe.Products) == 0 {
		return nil
	}
	unlockIndex, _ := dataClient.GetUnlockIndex()
	tierCtx := production.NewTierContext(hubTier, unlockIndex)
	cache := newRecipeLookupCache(dataClient)
	steps := aggregateChainSteps(collectChainSteps(cache, tierCtx, recipe.Products[0].ItemClassName, targetPerMin, 0, recipe))

	planSteps := make([]chainStep, 0, len(steps))
	for _, s := range steps {
		if s.skipPlan {
			continue
		}
		planSteps = append(planSteps, s)
	}
	if len(planSteps) == 0 {
		return nil
	}

	shardInputs := make([]production.ShardStepInput, len(planSteps))
	for i, s := range planSteps {
		shardInputs[i] = chainStepShardInput(s, tierCtx)
	}
	return &chainShardPlan{
		steps:   planSteps,
		budgets: production.DistributeShardBudgetOptimal(shardInputs, shards),
	}
}

func (p *chainShardPlan) budgetFor(itemClass, recipeClass string, raw bool) int {
	if p == nil {
		return 0
	}
	fallback := -1
	for i, s := range p.steps {
		if s.itemClass != itemClass || s.raw != raw {
			continue
		}
		if raw {
			return p.budgets[i]
		}
		if recipeClass != "" && s.recipe != nil && s.recipe.ClassName == recipeClass {
			return p.budgets[i]
		}
		if fallback < 0 {
			fallback = p.budgets[i]
		}
	}
	if fallback >= 0 {
		return fallback
	}
	return 0
}

func (p *chainShardPlan) rootBudget() int {
	if p == nil || len(p.budgets) == 0 {
		return 0
	}
	return p.budgets[len(p.budgets)-1]
}

func shardBudgetForChainItem(dataClient *clients.DataClient, rootRecipeClass string, rootRate float64, shards, hubTier int, itemClass, recipeClass string, raw bool) int {
	if rootRecipeClass == "" || shards <= 0 {
		return shards
	}
	plan := buildChainShardPlan(dataClient, rootRecipeClass, rootRate, shards, hubTier)
	if plan == nil {
		return shards
	}
	budget := plan.budgetFor(itemClass, recipeClass, raw)
	if budget > 0 {
		return budget
	}
	return 0
}

func buildFactoryPlan(dataClient *clients.DataClient, taskID int64, recipeClass string, targetPerMin float64, shards, hubTier int, logistics production.LogisticsParams) *production.FactoryPlan {
	recipe, err := dataClient.GetRecipe(recipeClass)
	if err != nil || recipe == nil || len(recipe.Products) == 0 {
		return nil
	}

	plan := &production.FactoryPlan{
		TaskID:       taskID,
		RecipeClass:  recipeClass,
		TargetAmount: targetPerMin,
	}

	shardPlan := buildChainShardPlan(dataClient, recipeClass, targetPerMin, shards, hubTier)
	if shardPlan == nil {
		return nil
	}
	planSteps := shardPlan.steps
	budgets := shardPlan.budgets

	ctx := loadProductionContext(dataClient, targetPerMin, targetPerMin, shards, hubTier, logistics)
	buildingCounts := map[string]struct {
		name  string
		count int
	}{}

	for i, s := range planSteps {
		var stepPlan *production.StepPlan
		if s.raw {
			stepPlan = ctx.planForRecipe(s.itemClass, s.itemName, s.amount, nil, true, budgets[i])
		} else {
			stepPlan = ctx.planForRecipe(s.itemClass, s.itemName, s.amount, s.recipe, false, budgets[i])
		}
		if stepPlan == nil || stepPlan.Chosen == nil {
			continue
		}
		plan.Steps = append(plan.Steps, production.FactoryStep{
			ItemName:      s.itemName,
			BuildingName:  stepPlan.BuildingName,
			BuildingClass: stepPlan.BuildingClass,
			Plan:          stepPlan,
			Chosen:        stepPlan.Chosen,
			BeltLines:     stepPlan.BeltLines,
		})
		plan.TotalShardsUsed += stepPlan.Chosen.ShardsUsed
		plan.TotalBuildings += stepPlan.Chosen.TotalMachines

		key := stepPlan.BuildingClass
		entry := buildingCounts[key]
		entry.name = stepPlan.BuildingName
		entry.count += stepPlan.Chosen.TotalMachines
		buildingCounts[key] = entry
	}

	for class, entry := range buildingCounts {
		costRow := production.BuildingCostRow{
			BuildingName:  entry.name,
			BuildingClass: class,
			Count:         entry.count,
		}
		buildRecipe, _ := dataClient.GetBuildingRecipe(class)
		if buildRecipe != nil {
			for _, ing := range buildRecipe.Ingredients {
				costRow.Materials = append(costRow.Materials, production.MaterialRow{
					ClassName:   ing.ItemClassName,
					DisplayName: itemDisplayNameFor(dataClient, ing.ItemClassName),
					IconURL:     clients.ItemIconURL(ing.ItemClassName),
					Amount:      round2(ing.Amount * float64(entry.count)),
				})
			}
		}
		plan.BuildingCosts = append(plan.BuildingCosts, costRow)
	}
	sort.Slice(plan.BuildingCosts, func(i, j int) bool {
		return plan.BuildingCosts[i].BuildingName < plan.BuildingCosts[j].BuildingName
	})

	nameFn := func(class string) string { return itemDisplayNameFor(dataClient, class) }
	plan.TotalMaterials = production.AggregateBuildingCosts(plan.BuildingCosts, nameFn)
	return plan
}

type productionSettings struct {
	Shards              int
	ConveyorMk          int
	PipeMk              int
	Logistics           production.LogisticsParams
	ShowFactoryDetails  bool
}

func productionSettingsFromTask(task clients.Task) productionSettings {
	shards := task.ProductionShards
	if shards < 0 {
		shards = 0
	}
	conveyorMk := production.NormalizeConveyorMk(task.ConveyorMk)
	pipeMk := production.NormalizePipeMk(task.PipeMk)
	logistics := production.LogisticsParams{ConveyorMk: conveyorMk, PipeMk: pipeMk}
	return productionSettings{
		Shards:             shards,
		ConveyorMk:         conveyorMk,
		PipeMk:             pipeMk,
		Logistics:          logistics,
		ShowFactoryDetails: logistics.Configured(),
	}
}

func parseProductionSettings(r *http.Request) productionSettings {
	_ = r.ParseForm()
	shards, _ := strconv.Atoi(firstNonEmpty(r.URL.Query().Get("shards"), r.FormValue("shards")))
	if shards < 0 {
		shards = 0
	}
	conveyorMk := production.NormalizeConveyorMk(atoiDefault(r.URL.Query().Get("conveyor_mk"), r.FormValue("conveyor_mk")))
	pipeMk := production.NormalizePipeMk(atoiDefault(r.URL.Query().Get("pipe_mk"), r.FormValue("pipe_mk")))
	calc := r.URL.Query().Get("calc") == "1" || r.FormValue("calc") == "1"
	logistics := production.LogisticsParams{ConveyorMk: conveyorMk, PipeMk: pipeMk}
	return productionSettings{
		Shards:             shards,
		ConveyorMk:         conveyorMk,
		PipeMk:             pipeMk,
		Logistics:          logistics,
		ShowFactoryDetails: calc && logistics.Configured(),
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func atoiDefault(a, b string) int {
	if v, err := strconv.Atoi(firstNonEmpty(a, b)); err == nil {
		return v
	}
	return 0
}

func parseShardCount(r *http.Request) int {
	return parseProductionSettings(r).Shards
}
