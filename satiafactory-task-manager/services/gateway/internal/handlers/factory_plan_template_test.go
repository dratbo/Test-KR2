package handlers

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"github.com/dratbo/satisfactory-task-manager/gateway/internal/production"
)

func TestFactoryPlanTemplateRendersConstructorCount(t *testing.T) {
	tmpl, err := template.ParseFiles("../../templates/factory_plan.html")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	chosenCtor := production.Scenario{
		Name:          "Без энергомодулей",
		TotalMachines: 5,
		TotalRate:     90,
		Slots: []production.MachineSlot{
			{Count: 5, ClockPercent: 100, RatePerMachine: 20},
		},
	}
	plan := &production.FactoryPlan{
		TargetAmount: 90,
		Steps: []production.FactoryStep{
			{
				ItemName:     "Iron Plate",
				BuildingName: "Конструктор",
				Chosen:       &chosenCtor,
			},
		},
		TotalBuildings: 5,
		BuildingCosts: []production.BuildingCostRow{
			{BuildingName: "Конструктор", Count: 5},
		},
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, plan); err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Конструктор") {
		t.Fatalf("missing constructor:\n%s", out)
	}
	if !strings.Contains(out, "5 шт") {
		t.Fatalf("missing constructor count:\n%s", out)
	}
}
