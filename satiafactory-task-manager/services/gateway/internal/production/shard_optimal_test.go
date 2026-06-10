package production

import "testing"

func TestDistributeShardBudgetOptimalSteelChain(t *testing.T) {
	// Steel beam 210/min: miners 840 @ 60, foundry 840 @ 45, constructor 210 @ 15.
	steps := []ShardStepInput{
		{RequiredRate: 840, BaseRate: 60, Overclockable: true},
		{RequiredRate: 840, BaseRate: 60, Overclockable: true},
		{RequiredRate: 840, BaseRate: 45, Overclockable: true},
		{RequiredRate: 210, BaseRate: 15, Overclockable: true},
	}
	budgets := DistributeShardBudgetOptimal(steps, 10)
	totalUsed := TotalShardsUsedForBudget(steps, budgets)
	if totalUsed != 10 {
		t.Fatalf("expected 10 shards used across chain, got %d (budgets %v)", totalUsed, budgets)
	}
	foundrySlots := AllocateWithShardBudget(steps[2].RequiredRate, steps[2].BaseRate, budgets[2])
	if TotalShardsUsed(foundrySlots) == 0 {
		t.Fatalf("foundry should use shards, budgets %v", budgets)
	}
}

func TestDistributeShardBudgetOptimalSpendsAllAvailable(t *testing.T) {
	steps := []ShardStepInput{
		{RequiredRate: 840.1, BaseRate: 45, Overclockable: true},
		{RequiredRate: 210, BaseRate: 15, Overclockable: true},
	}
	budgets := DistributeShardBudgetOptimal(steps, 10)
	if TotalShardsUsedForBudget(steps, budgets) != 10 {
		t.Fatalf("expected all 10 shards used, got budgets %v", budgets)
	}
}

func TestShardGainFoundryBudget4To5(t *testing.T) {
	before := AllocateWithShardBudget(840, 45, 4)
	after := AllocateWithShardBudget(840, 45, 5)
	if TotalShardsUsed(after) <= TotalShardsUsed(before) {
		t.Fatalf("expected more shards used with budget 5, before=%d after=%d",
			TotalShardsUsed(before), TotalShardsUsed(after))
	}
}
