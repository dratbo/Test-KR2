package production

// FactoryStep is one production step in the full factory chain.
type FactoryStep struct {
	ItemName      string
	BuildingName  string
	BuildingClass string
	Plan          *StepPlan
	Chosen        *Scenario
}

// FactoryPlan is the full production and construction plan for a task.
type FactoryPlan struct {
	TaskID          int64
	RecipeClass     string
	TargetAmount    float64
	Steps           []FactoryStep
	BuildingCosts   []BuildingCostRow
	TotalMaterials  []MaterialRow
	TotalShardsUsed int
	TotalBuildings  int
}
