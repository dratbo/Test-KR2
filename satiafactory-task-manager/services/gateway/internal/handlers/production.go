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
}

func loadProductionContext(dataClient *clients.DataClient, rootTotalItems, rootRate float64, availableShards int) productionContext {
	buildings, _ := dataClient.GetBuildings()
	return productionContext{
		lookup:          production.BuildingLookupFromList(buildings),
		rootTotalItems:  rootTotalItems,
		rootRate:        rootRate,
		availableShards: availableShards,
	}
}

func (ctx productionContext) planForRecipe(itemClass, itemName string, amount float64, recipe *clients.Recipe, raw bool, shardBudget int) *production.StepPlan {
	requiredRate := amount
	if ctx.rootTotalItems > 0 && ctx.rootRate > 0 {
		requiredRate = production.IngredientRequiredRate(amount, ctx.rootTotalItems, ctx.rootRate)
	}
	return production.BuildStepPlan(production.PlanInput{
		ItemClass:       itemClass,
		ItemName:        itemName,
		TotalItems:      amount,
		RequiredRate:    requiredRate,
		Recipe:          recipe,
		RawResource:     raw,
		BuildingLookup:  ctx.lookup,
		ShardBudget:     shardBudget,
		AvailableShards: ctx.availableShards,
	})
}

func planForTaskRecipe(dataClient *clients.DataClient, recipe *clients.Recipe, targetPerMin float64, availableShards, shardBudget int) *production.StepPlan {
	if recipe == nil || len(recipe.Products) == 0 {
		return nil
	}
	productAmount := recipe.Products[0].Amount
	params := production.RootPlanParamsFromTask(targetPerMin, productAmount)

	prodClass := recipe.Products[0].ItemClassName
	itemName := itemDisplayName(dataClient, prodClass)

	ctx := loadProductionContext(dataClient, params.RequiredRate, params.RequiredRate, availableShards)
	return ctx.planForRecipe(prodClass, itemName, params.RequiredRate, recipe, false, shardBudget)
}

type chainStep struct {
	itemClass string
	itemName  string
	amount    float64
	recipe    *clients.Recipe
	raw       bool
}

func itemDisplayNameFor(dataClient *clients.DataClient, itemClass string) string {
	return itemDisplayName(dataClient, itemClass)
}

func collectChainSteps(dataClient *clients.DataClient, itemClass string, amount float64, depth int) []chainStep {
	if depth >= maxChainTreeDepth {
		return []chainStep{{
			itemClass: itemClass,
			itemName:  itemDisplayNameFor(dataClient, itemClass),
			amount:    amount,
			raw:       true,
		}}
	}
	recipes, err := dataClient.GetRecipesByProduct(itemClass, false)
	if err != nil || len(recipes) == 0 {
		return []chainStep{{
			itemClass: itemClass,
			itemName:  itemDisplayNameFor(dataClient, itemClass),
			amount:    amount,
			raw:       true,
		}}
	}
	recipe := &recipes[0]
	prodAmount := 1.0
	if len(recipe.Products) > 0 {
		prodAmount = recipe.Products[0].Amount
	}

	var steps []chainStep
	for _, ing := range recipe.Ingredients {
		ingAmount := round2(ing.Amount * amount / prodAmount)
		steps = append(steps, collectChainSteps(dataClient, ing.ItemClassName, ingAmount, depth+1)...)
	}
	steps = append(steps, chainStep{
		itemClass: itemClass,
		itemName:  itemDisplayNameFor(dataClient, itemClass),
		amount:    amount,
		recipe:    recipe,
		raw:       false,
	})
	return steps
}

func buildFactoryPlan(dataClient *clients.DataClient, taskID int64, recipeClass string, targetPerMin float64, shards int) *production.FactoryPlan {
	recipe, err := dataClient.GetRecipe(recipeClass)
	if err != nil || recipe == nil || len(recipe.Products) == 0 {
		return nil
	}

	plan := &production.FactoryPlan{
		TaskID:       taskID,
		RecipeClass:  recipeClass,
		TargetAmount: targetPerMin,
	}

	steps := collectChainSteps(dataClient, recipe.Products[0].ItemClassName, targetPerMin, 0)
	if len(steps) == 0 {
		return nil
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

	ctx := loadProductionContext(dataClient, targetPerMin, targetPerMin, shards)
	buildingCounts := map[string]struct {
		name  string
		count int
	}{}

	for i, s := range steps {
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

func parseShardCount(r *http.Request) int {
	shards, _ := strconv.Atoi(r.URL.Query().Get("shards"))
	if shards == 0 {
		_ = r.ParseForm()
		shards, _ = strconv.Atoi(r.FormValue("shards"))
	}
	if shards < 0 {
		return 0
	}
	return shards
}
