package production

import (
	"testing"

	"github.com/dratbo/satisfactory-task-manager/gateway/internal/clients"
)

func TestPickExtractorRespectsHubTier(t *testing.T) {
	idx := &clients.UnlockIndex{
		BuildingTiers: map[string]int{
			"Build_MinerMk1_C": 0,
			"Build_MinerMk2_C": 4,
			"Build_MinerMk3_C": 8,
		},
	}
	tier3 := NewTierContext(3, idx)
	ext := PickExtractor("Desc_OreIron_C", tier3)
	if ext.BuildingClass != "Build_MinerMk1_C" {
		t.Fatalf("tier 3 expected Mk1 miner, got %s", ext.BuildingClass)
	}

	tier8 := NewTierContext(8, idx)
	ext8 := PickExtractor("Desc_OreIron_C", tier8)
	if ext8.BuildingClass != "Build_MinerMk3_C" {
		t.Fatalf("tier 8 expected Mk3 miner, got %s", ext8.BuildingClass)
	}
}

func TestRecipeMinTier(t *testing.T) {
	idx := &clients.UnlockIndex{
		RecipeTiers: map[string]int{
			"Recipe_MotorTurbo_C": 8,
		},
	}
	tierCtx := NewTierContext(3, idx)
	if tierCtx.RecipeUnlocked("Recipe_MotorTurbo_C") {
		t.Fatal("turbo motor should be locked at tier 3")
	}
	if tierCtx.RecipeMinTier("Recipe_MotorTurbo_C") != 8 {
		t.Fatalf("expected min tier 8, got %d", tierCtx.RecipeMinTier("Recipe_MotorTurbo_C"))
	}
}

func TestFilterChainRecipesByTier(t *testing.T) {
	recipes := []clients.Recipe{
		{ClassName: "Recipe_HadronCollider_C", ProducedIn: []string{"Desc_HadronCollider_C"}},
		{ClassName: "Recipe_ConstructorMk1_C", ProducedIn: []string{"Desc_ConstructorMk1_C"}, ManufactoringMenuPriority: 10},
	}
	idx := &clients.UnlockIndex{
		RecipeTiers: map[string]int{
			"Recipe_HadronCollider_C": 8,
			"Recipe_ConstructorMk1_C": 0,
		},
		BuildingTiers: map[string]int{
			"Build_HadronCollider_C": 8,
			"Build_ConstructorMk1_C": 0,
		},
	}
	tier5 := NewTierContext(5, idx)
	filtered := FilterChainRecipes(recipes, tier5)
	if len(filtered) != 1 || filtered[0].ClassName != "Recipe_ConstructorMk1_C" {
		t.Fatalf("expected only constructor at tier 5, got %v", filtered)
	}
}
