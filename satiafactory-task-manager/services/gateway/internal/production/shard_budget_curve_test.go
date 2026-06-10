package production

import "testing"

func TestAllocateWithShardBudgetFoundryCurve(t *testing.T) {
	prevUsed := 0
	for b := 1; b <= 12; b++ {
		slots := AllocateWithShardBudget(840.1, 45, b)
		used := TotalShardsUsed(slots)
		if used < prevUsed {
			t.Fatalf("budget %d: used %d < prev %d", b, used, prevUsed)
		}
		prevUsed = used
		if b == 10 && used < 10 {
			t.Fatalf("budget 10 should use 10 shards, got %d", used)
		}
	}
}
