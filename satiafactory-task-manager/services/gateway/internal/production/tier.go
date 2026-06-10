package production

import (
	"sort"
	"strconv"

	"github.com/dratbo/satisfactory-task-manager/gateway/internal/clients"
)

// TierContext caps available buildings and recipes by HUB milestone tier.
type TierContext struct {
	MaxTier       int
	RecipeTiers   map[string]int
	BuildingTiers map[string]int
}

func NormalizeHubTier(tier int) int {
	if tier <= 0 {
		return 9
	}
	if tier > 9 {
		return 9
	}
	return tier
}

func NewTierContext(maxTier int, index *clients.UnlockIndex) *TierContext {
	ctx := &TierContext{
		MaxTier:       NormalizeHubTier(maxTier),
		RecipeTiers:   map[string]int{},
		BuildingTiers: map[string]int{},
	}
	if index == nil {
		return ctx
	}
	for k, v := range index.RecipeTiers {
		ctx.RecipeTiers[k] = v
	}
	for k, v := range index.BuildingTiers {
		ctx.BuildingTiers[k] = v
	}
	return ctx
}

func (c *TierContext) RecipeUnlocked(recipeClass string) bool {
	if c == nil || recipeClass == "" {
		return true
	}
	return c.RecipeMinTier(recipeClass) <= c.MaxTier
}

// RecipeMinTier returns the HUB tier that unlocks a recipe (0 = from the start).
func (c *TierContext) RecipeMinTier(recipeClass string) int {
	if c == nil || recipeClass == "" {
		return 0
	}
	if minTier, ok := c.RecipeTiers[recipeClass]; ok {
		return minTier
	}
	return 0
}

func (c *TierContext) BuildingUnlocked(buildingClass string) bool {
	if c == nil || buildingClass == "" {
		return true
	}
	buildingClass = normalizeBuildingClass(buildingClass)
	minTier, ok := c.BuildingTiers[buildingClass]
	if !ok {
		return true
	}
	return minTier <= c.MaxTier
}

func (c *TierContext) RecipeBuildingUnlocked(recipe *clients.Recipe) bool {
	if recipe == nil {
		return true
	}
	buildingClass := PickFactoryBuilding(recipe.ProducedIn)
	if buildingClass == "" {
		return true
	}
	return c.BuildingUnlocked(buildingClass)
}

// HubTierLabel returns a short Russian label for UI.
func HubTierLabel(tier int) string {
	tier = NormalizeHubTier(tier)
	return "Тир HUB " + strconv.Itoa(tier)
}

// solidMinerOptions ordered best to worst.
var solidMinerOptions = []Extractor{
	{BuildingClass: "Build_MinerMk3_C", BaseRate: 240},
	{BuildingClass: "Build_MinerMk2_C", BaseRate: 120},
	{BuildingClass: "Build_MinerMk1_C", BaseRate: 60},
}

var fluidExtractorOptions = map[string][]Extractor{
	"Desc_LiquidOil_C": {
		{BuildingClass: "Build_OilPump_C", BaseRate: 120},
	},
	"Desc_Water_C": {
		{BuildingClass: "Build_WaterExtractor_C", BaseRate: 120},
	},
	"Desc_NitrogenGas_C": {
		{BuildingClass: "Build_FrackingExtractor_C", BaseRate: 120},
	},
}

func pickBestExtractor(itemClass string, tier *TierContext) Extractor {
	if options, ok := fluidExtractorOptions[itemClass]; ok {
		for _, opt := range options {
			if tier == nil || tier.BuildingUnlocked(opt.BuildingClass) {
				return opt
			}
		}
		if len(options) > 0 {
			return options[len(options)-1]
		}
	}
	for _, opt := range solidMinerOptions {
		if tier == nil || tier.BuildingUnlocked(opt.BuildingClass) {
			return opt
		}
	}
	return solidMinerOptions[len(solidMinerOptions)-1]
}

// FilterChainRecipes keeps only recipes unlocked at the task tier.
func FilterChainRecipes(recipes []clients.Recipe, tier *TierContext) []clients.Recipe {
	if tier == nil {
		return recipes
	}
	out := make([]clients.Recipe, 0, len(recipes))
	for _, r := range recipes {
		if !tier.RecipeUnlocked(r.ClassName) {
			continue
		}
		if !tier.RecipeBuildingUnlocked(&r) {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ManufactoringMenuPriority > out[j].ManufactoringMenuPriority
	})
	return out
}

// PickChainRecipeWithTier chooses the best recipe allowed at the given tier.
func PickChainRecipeWithTier(recipes []clients.Recipe, tier *TierContext) *clients.Recipe {
	filtered := FilterChainRecipes(recipes, tier)
	if len(filtered) == 0 {
		return PickChainRecipe(recipes)
	}
	return PickChainRecipe(filtered)
}

// TierOptions returns selectable HUB tiers for forms.
func TierOptions() []struct {
	Value int
	Label string
} {
	return []struct {
		Value int
		Label string
	}{
		{1, "Тир 1 — Основы"},
		{2, "Тир 2 — Кварц"},
		{3, "Тир 3 — Угольная энергия"},
		{4, "Тир 4 — Сталь"},
		{5, "Тир 5 — Промышленность"},
		{6, "Тир 6 — Платформа"},
		{7, "Тир 7 — Алюминий"},
		{8, "Тир 8 — Ядерная программа"},
		{9, "Тир 9 — Космическая программа"},
	}
}

// BuildingRequiresHigherTier reports if the building needs a higher HUB tier.
func (c *TierContext) BuildingRequiresHigherTier(buildingClass string) (bool, int) {
	buildingClass = normalizeBuildingClass(buildingClass)
	minTier, ok := c.BuildingTiers[buildingClass]
	if !ok {
		return false, 0
	}
	if minTier <= c.MaxTier {
		return false, minTier
	}
	return true, minTier
}
