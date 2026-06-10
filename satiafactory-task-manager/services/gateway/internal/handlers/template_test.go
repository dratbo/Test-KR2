package handlers

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"github.com/dratbo/satisfactory-task-manager/gateway/internal/clients"
	"github.com/dratbo/satisfactory-task-manager/gateway/internal/production"
)

func TestTaskDetailTemplateRendersIngredients(t *testing.T) {
	funcMap := template.FuncMap{
		"formatItem":  formatItemName,
		"statusLabel": statusLabel,
	}
	tmpl, err := template.New("task_detail.html").Funcs(funcMap).ParseFiles(
		"../../templates/task_detail.html",
		"../../templates/task_production.html",
		"../../templates/factory_plan.html",
		"../../templates/production_plan.html",
	)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	chosen := production.Scenario{Name: "test", TotalMachines: 5, TotalRate: 90}
	plan := &production.FactoryPlan{
		TargetAmount:   90,
		Steps:          []production.FactoryStep{{ItemName: "Iron Plate", BuildingName: "Конструктор", Chosen: &chosen}},
		TotalBuildings: 5,
	}
	stepPlan := &production.StepPlan{
		BuildingName: "Конструктор",
		BaseRate:     20,
		RequiredRate: 90,
		Chosen:       &chosen,
		Scenarios:    []production.Scenario{chosen},
	}

	data := struct {
		Task           TaskView
		Users          []clients.UserBrief
		FactoryPlan    *production.FactoryPlan
		ProductionData taskProductionData
	}{
		Task: TaskView{
			ID:                  1,
			Title:               "Test",
			RecipeName:          "Iron Plate",
			TargetAmount:        90,
			TargetItemClassName: "Recipe_IronPlate_C",
			Ingredients: []ingredientRow{
				{Name: "Iron Ingot", Class: "Desc_IronIngot_C", Amount: 135, IconURL: "/icons/x.png", Craftable: true},
			},
			Products: []ingredientRow{
				{Name: "Iron Plate", Amount: 90},
			},
		},
		ProductionData: taskProductionData{
			TaskID:             1,
			HubTier:            9,
			HubTierLabel:       "Тир HUB 9",
			AvailableShards:    0,
			ConveyorMk:         3,
			PipeMk:             1,
			ShowFactoryDetails: true,
			FactoryPlan:        plan,
			ProductionPlan:     stepPlan,
		},
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Нужно для изготовления") {
		t.Fatalf("missing ingredients section:\n%s", out)
	}
	if !strings.Contains(out, "Iron Ingot") {
		t.Fatalf("missing ingredient row:\n%s", out)
	}
	if !strings.Contains(out, "Конструктор") {
		t.Fatalf("missing factory plan:\n%s", out)
	}
}
