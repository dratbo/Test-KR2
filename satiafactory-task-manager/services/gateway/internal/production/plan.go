package production

import (
	"strings"

	"github.com/dratbo/satisfactory-task-manager/gateway/internal/clients"
)

// StepPlan is production planning data for one item in the chain.
type StepPlan struct {
	ItemName         string
	ItemClass        string
	BuildingName     string
	BuildingClass    string
	IsExtractor      bool
	AllowsPowerShard bool
	ProductPerCycle  float64
	Duration         float64
	BaseRate         float64
	RequiredRate     float64
	TotalItems       float64
	RateTable        []RateTableEntry
	Scenarios        []Scenario
	Chosen           *Scenario
	ShardBudget      int
	AvailableShards  int
	BeltLines        int
}

// PlanInput configures production plan generation.
type PlanInput struct {
	ItemClass       string
	ItemName        string
	TotalItems      float64
	RequiredRate    float64
	Recipe          *clients.Recipe
	RawResource     bool
	BuildingLookup  func(className string) string
	ShardBudget     int
	AvailableShards int
	Tier            *TierContext
	Logistics       LogisticsParams
}

// BuildStepPlan creates a production plan for one chain step.
func BuildStepPlan(in PlanInput) *StepPlan {
	if in.TotalItems <= 0 && in.RequiredRate <= 0 {
		return nil
	}

	plan := &StepPlan{
		ItemName:        in.ItemName,
		ItemClass:       in.ItemClass,
		RequiredRate:    round2(in.RequiredRate),
		TotalItems:      round2(in.TotalItems),
		ShardBudget:     in.ShardBudget,
		AvailableShards: in.AvailableShards,
	}

	if in.RawResource || in.Recipe == nil {
		ext := PickExtractor(in.ItemClass, in.Tier)
		plan.IsExtractor = true
		plan.BuildingClass = ext.BuildingClass
		plan.BuildingName = BuildingDisplayName(ext.BuildingClass, "")
		if in.BuildingLookup != nil {
			if name := in.BuildingLookup(ext.BuildingClass); name != "" {
				plan.BuildingName = BuildingDisplayName(ext.BuildingClass, name)
			}
		}
		plan.BaseRate = ext.BaseRate
		plan.AllowsPowerShard = SupportsPowerShards(ext.BuildingClass)
		plan.RateTable = BuildRateTable(0, 0, ext.BaseRate)
		plan.Scenarios = BuildScenarios(in.RequiredRate, ext.BaseRate, plan.AllowsPowerShard, in.ShardBudget)
		plan.Chosen = PickScenario(plan.Scenarios, in.ShardBudget)
		applyLogisticsCap(plan, in)
		return plan
	}

	productAmount := 0.0
	if len(in.Recipe.Products) > 0 {
		productAmount = in.Recipe.Products[0].Amount
	}
	plan.ProductPerCycle = productAmount
	plan.Duration = in.Recipe.Duration
	plan.BaseRate = ItemsPerMinute(productAmount, in.Recipe.Duration, 100)

	buildingClass := PickFactoryBuilding(in.Recipe.ProducedIn)
	if in.Tier != nil && buildingClass != "" && !in.Tier.BuildingUnlocked(buildingClass) {
		return nil
	}
	englishName := ""
	if in.BuildingLookup != nil && buildingClass != "" {
		englishName = in.BuildingLookup(buildingClass)
	}
	if buildingClass == "" {
		return nil
	}

	plan.BuildingClass = buildingClass
	plan.BuildingName = BuildingDisplayName(buildingClass, englishName)
	plan.AllowsPowerShard = SupportsPowerShards(buildingClass)
	plan.RateTable = BuildRateTable(productAmount, in.Recipe.Duration, plan.BaseRate)
	plan.Scenarios = BuildScenarios(in.RequiredRate, plan.BaseRate, plan.AllowsPowerShard, in.ShardBudget)
	plan.Chosen = PickScenario(plan.Scenarios, in.ShardBudget)
	applyLogisticsCap(plan, in)
	return plan
}

func applyLogisticsCap(plan *StepPlan, in PlanInput) {
	if plan == nil || plan.Chosen == nil || !in.Logistics.Configured() {
		return
	}
	capacity := in.Logistics.BeltRate()
	if IsFluidItem(in.ItemClass) {
		capacity = in.Logistics.PipeRate()
	}
	capped := ApplyBeltCap(plan.Chosen.Slots, capacity)
	plan.BeltLines = len(capped)
	plan.Chosen.Slots = capped
	plan.Chosen.TotalMachines = TotalMachines(capped)
	plan.Chosen.TotalRate = round2(TotalRate(capped))
}

// RootPlanParams holds parameters for the main task product.
type RootPlanParams struct {
	TargetAmount   float64
	ProductAmount  float64
	TotalItems     float64
	RequiredRate   float64
}

// RootPlanParamsFromTask derives planning rates from target production (items/min).
func RootPlanParamsFromTask(targetPerMin, productPerCycle float64) RootPlanParams {
	if targetPerMin <= 0 {
		targetPerMin = 1
	}
	return RootPlanParams{
		TargetAmount:  targetPerMin,
		ProductAmount: productPerCycle,
		TotalItems:    round2(targetPerMin),
		RequiredRate:  round2(targetPerMin),
	}
}

// IngredientRequiredRate computes upstream throughput for an ingredient row.
func IngredientRequiredRate(ingredientAmount, rootTotalItems, rootRequiredRate float64) float64 {
	if rootTotalItems <= 0 {
		return 0
	}
	return round2(ingredientAmount * rootRequiredRate / rootTotalItems)
}

// BuildingLookupFromList creates a lookup function from API building list.
func BuildingLookupFromList(buildings []clients.Building) func(string) string {
	m := make(map[string]string, len(buildings))
	for _, b := range buildings {
		m[b.ClassName] = b.DisplayName
	}
	return func(className string) string {
		return m[className]
	}
}

// IsAlternateRecipe reports alternate recipe class names.
func IsAlternateRecipe(className, displayName string) bool {
	return strings.Contains(className, "Alternate") || strings.HasPrefix(displayName, "Alternate:")
}
