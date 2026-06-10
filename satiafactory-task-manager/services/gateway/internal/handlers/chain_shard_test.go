package handlers

import (
	"testing"

	"github.com/dratbo/satisfactory-task-manager/gateway/internal/clients"
	"github.com/dratbo/satisfactory-task-manager/gateway/internal/production"
)

func TestChainShardPlanBudgetForIronPlateChain(t *testing.T) {
	shardInputs := []production.ShardStepInput{
		{RequiredRate: 135, BaseRate: 60, Overclockable: true},
		{RequiredRate: 135, BaseRate: 15, Overclockable: true},
		{RequiredRate: 90, BaseRate: 20, Overclockable: true},
	}
	budgets := production.DistributeShardBudgetOptimal(shardInputs, 4)
	plan := &chainShardPlan{
		steps: []chainStep{
			{itemClass: "Desc_OreIron_C", raw: true},
			{itemClass: "Desc_IronIngot_C", recipe: &clients.Recipe{ClassName: "Recipe_IronIngot_C"}, raw: false},
			{itemClass: "Desc_IronPlate_C", recipe: &clients.Recipe{ClassName: "Recipe_IronPlate_C"}, raw: false},
		},
		budgets: budgets,
	}

	if production.TotalShardsUsedForBudget(shardInputs, budgets) != 4 {
		t.Fatalf("expected 4 shards used, got budgets %v", budgets)
	}

	ingotBudget := plan.budgetFor("Desc_IronIngot_C", "Recipe_IronIngot_C", false)
	if ingotBudget == 0 {
		t.Fatalf("smelter step should receive shard budget, got %v", budgets)
	}
}
