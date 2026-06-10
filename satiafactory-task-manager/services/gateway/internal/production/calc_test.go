package production

import (
	"math"
	"testing"
)

func TestItemsPerMinuteIronPlate(t *testing.T) {
	rate := ItemsPerMinute(2, 6, 100)
	if math.Abs(rate-20) > 0.01 {
		t.Fatalf("expected 20 items/min, got %v", rate)
	}
	rate250 := ItemsPerMinute(2, 6, 250)
	if math.Abs(rate250-50) > 0.01 {
		t.Fatalf("expected 50 items/min at 250%%, got %v", rate250)
	}
}

func TestAllocateWithoutShards90(t *testing.T) {
	slots := AllocateWithoutShards(90, 20)
	total := TotalMachines(slots)
	if total != 5 {
		t.Fatalf("expected 5 machines, got %d (%v)", total, slots)
	}
	if math.Abs(TotalRate(slots)-90) > 0.1 {
		t.Fatalf("expected total rate 90, got %v", TotalRate(slots))
	}
}

func TestAllocateWithShards90(t *testing.T) {
	slots := AllocateWithShards(90, 20)
	total := TotalMachines(slots)
	if total != 2 {
		t.Fatalf("expected 2 machines, got %d (%v)", total, slots)
	}
	if math.Abs(TotalRate(slots)-90) > 0.1 {
		t.Fatalf("expected total rate 90, got %v", TotalRate(slots))
	}
}

func TestAllocateWithShardBudget3(t *testing.T) {
	slots := AllocateWithShardBudget(90, 20, 3)
	if TotalShardsUsed(slots) > 3 {
		t.Fatalf("expected at most 3 shards used, got %d", TotalShardsUsed(slots))
	}
	if math.Abs(TotalRate(slots)-90) > 0.1 {
		t.Fatalf("expected total rate 90, got %v", TotalRate(slots))
	}
}

func TestIngredientRequiredRate(t *testing.T) {
	rate := IngredientRequiredRate(135, 90, 90)
	if math.Abs(rate-135) > 0.01 {
		t.Fatalf("expected 135 ingots/min, got %v", rate)
	}
}

func TestDistributeShardBudget(t *testing.T) {
	budgets := DistributeShardBudget(3, []bool{true, true, true}, 5)
	sum := 0
	for _, b := range budgets {
		sum += b
	}
	if sum != 5 {
		t.Fatalf("expected total budget 5, got %v", budgets)
	}
}
