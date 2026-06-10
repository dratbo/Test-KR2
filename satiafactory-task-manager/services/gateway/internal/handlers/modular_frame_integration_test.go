//go:build integration

package handlers

import (
	"os"
	"strings"
	"testing"

	"github.com/dratbo/satisfactory-task-manager/gateway/internal/clients"
	"github.com/dratbo/satisfactory-task-manager/gateway/internal/production"
)

func TestModularFrameFactoryPlan150(t *testing.T) {
	baseURL := os.Getenv("DATA_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8083"
	}
	dc := clients.NewDataClient(baseURL)

	plan := buildFactoryPlan(dc, 1, "Recipe_ModularFrame_C", 150, 10, 9, production.LogisticsParams{ConveyorMk: 4, PipeMk: 1})
	if plan == nil {
		t.Fatal("factory plan is nil")
	}
	if len(plan.Steps) == 0 {
		t.Fatal("factory plan has no steps")
	}
	if plan.TotalShardsUsed != 10 {
		t.Fatalf("expected 10 shards used, got %d", plan.TotalShardsUsed)
	}
	stepsWithShards := 0
	for _, step := range plan.Steps {
		if step.Chosen != nil && step.Chosen.ShardsUsed > 0 {
			stepsWithShards++
		}
	}
	if stepsWithShards < 2 {
		t.Fatalf("expected shards spread across chain, got %d steps with shards", stepsWithShards)
	}
	for _, step := range plan.Steps {
		if step.Chosen == nil {
			t.Fatalf("step %s has no chosen scenario", step.ItemName)
		}
		if step.Chosen.TotalRate+0.5 < step.Plan.RequiredRate {
			t.Fatalf("step %s: rate %.1f < required %.1f", step.ItemName, step.Chosen.TotalRate, step.Plan.RequiredRate)
		}
	}
	t.Logf("steps=%d buildings=%d shards=%d", len(plan.Steps), plan.TotalBuildings, plan.TotalShardsUsed)
	for _, step := range plan.Steps {
		shards := 0
		if step.Chosen != nil {
			shards = step.Chosen.ShardsUsed
		}
		t.Logf("  %s: %d machines, %d shards, required %.1f", step.ItemName, step.Chosen.TotalMachines, shards, step.Plan.RequiredRate)
	}
}

func TestModularFrameFactoryPlanTier1MissingAssembler(t *testing.T) {
	baseURL := os.Getenv("DATA_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8083"
	}
	dc := clients.NewDataClient(baseURL)
	plan := buildFactoryPlan(dc, 1, "Recipe_ModularFrame_C", 150, 10, 1, production.LogisticsParams{ConveyorMk: 3, PipeMk: 1})
	if plan == nil {
		t.Fatal("factory plan is nil")
	}
	hasMF := false
	for _, step := range plan.Steps {
		if strings.Contains(step.ItemName, "Modular") || strings.Contains(step.ItemName, "каркас") {
			hasMF = true
		}
	}
	if hasMF {
		t.Fatal("tier 1 should not include modular frame assembler step")
	}
}

func TestModularFrameFactoryPlanTier2(t *testing.T) {
	baseURL := os.Getenv("DATA_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8083"
	}
	dc := clients.NewDataClient(baseURL)
	plan := buildFactoryPlan(dc, 1, "Recipe_ModularFrame_C", 150, 10, 2, production.LogisticsParams{ConveyorMk: 3, PipeMk: 1})
	if plan == nil {
		t.Fatal("factory plan is nil")
	}
	if plan.TotalShardsUsed != 10 {
		t.Fatalf("tier2: expected 10 shards used, got %d", plan.TotalShardsUsed)
	}
	for _, step := range plan.Steps {
		if step.Chosen != nil && step.Chosen.TotalRate+0.5 < step.Plan.RequiredRate {
			t.Fatalf("tier2 step %s: rate %.1f < required %.1f", step.ItemName, step.Chosen.TotalRate, step.Plan.RequiredRate)
		}
	}
}

func TestModularFrameFactoryPlan4Shards(t *testing.T) {
	baseURL := os.Getenv("DATA_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8083"
	}
	dc := clients.NewDataClient(baseURL)
	plan := buildFactoryPlan(dc, 1, "Recipe_ModularFrame_C", 150, 4, 9, production.LogisticsParams{ConveyorMk: 4, PipeMk: 1})
	if plan == nil {
		t.Fatal("factory plan is nil")
	}
	if plan.TotalShardsUsed != 4 {
		t.Fatalf("expected 4 shards used, got %d", plan.TotalShardsUsed)
	}
}

func TestModularFrameShardBudgetsPerStep(t *testing.T) {
	baseURL := os.Getenv("DATA_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8083"
	}
	dc := clients.NewDataClient(baseURL)

	shardPlan := buildChainShardPlan(dc, "Recipe_ModularFrame_C", 150, 10, 2)
	if shardPlan == nil {
		t.Fatal("shard plan is nil")
	}
	sum := production.TotalShardsUsedForBudget(shardInputsFromPlan(dc, shardPlan, 2), shardPlan.budgets)
	if sum != 10 {
		t.Fatalf("expected 10 shards in budgets, got %d budgets=%v", sum, shardPlan.budgets)
	}
	ripBudget := shardPlan.budgetFor("Desc_IronPlateReinforced_C", "Recipe_IronPlateReinforced_C", false)
	mfBudget := shardPlan.rootBudget()
	t.Logf("RIP budget=%d root budget=%d", ripBudget, mfBudget)
}

func TestModularFrameIngredientChainRates(t *testing.T) {
	baseURL := os.Getenv("DATA_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8083"
	}
	dc := clients.NewDataClient(baseURL)
	logistics := production.LogisticsParams{ConveyorMk: 4, PipeMk: 1}
	ctx := loadProductionContext(dc, 150, 150, 10, 2, logistics)

	ripRecipe, _ := dc.GetRecipe("Recipe_IronPlateReinforced_C")
	plan := ctx.planForRecipe("Desc_IronPlateReinforced_C", "RIP", 225, ripRecipe, false, 0)
	if plan == nil {
		t.Fatal("RIP plan is nil")
	}
	if plan.RequiredRate != 225 {
		t.Fatalf("RIP required rate: got %.1f want 225", plan.RequiredRate)
	}

	rodRecipe, _ := dc.GetRecipe("Recipe_IronRod_C")
	rodPlan := ctx.planForRecipe("Desc_IronRod_C", "Rod", 900, rodRecipe, false, 0)
	if rodPlan == nil || rodPlan.RequiredRate != 900 {
		t.Fatalf("rod required rate: got %v", rodPlan)
	}
}

func TestModularFrameSpendsAllShardsCounts(t *testing.T) {
	baseURL := os.Getenv("DATA_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8083"
	}
	dc := clients.NewDataClient(baseURL)
	shardPlan := buildChainShardPlan(dc, "Recipe_ModularFrame_C", 150, 10, 9)
	if shardPlan == nil {
		t.Fatal("shard plan is nil")
	}
	inputs := shardInputsFromPlan(dc, shardPlan, 9)
	for total := 1; total <= 15; total++ {
		budgets := production.DistributeShardBudgetOptimal(inputs, total)
		used := production.TotalShardsUsedForBudget(inputs, budgets)
		if used != total {
			t.Fatalf("shards=%d: used %d budgets=%v", total, used, budgets)
		}
	}
}

func shardInputsFromPlan(dc *clients.DataClient, plan *chainShardPlan, hubTier int) []production.ShardStepInput {
	unlockIndex, _ := dc.GetUnlockIndex()
	tierCtx := production.NewTierContext(hubTier, unlockIndex)
	out := make([]production.ShardStepInput, len(plan.steps))
	for i, s := range plan.steps {
		out[i] = chainStepShardInput(s, tierCtx)
	}
	return out
}
