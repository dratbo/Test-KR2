package parser

import "testing"

func TestDescToBuildClass(t *testing.T) {
	if got := descToBuildClass("Desc_ConstructorMk1_C"); got != "Build_ConstructorMk1_C" {
		t.Fatalf("constructor: %s", got)
	}
	if got := descToBuildClass("Desc_Converter_C"); got != "Build_Converter_C" {
		t.Fatalf("converter: %s", got)
	}
}

func TestNormalizeProducedInClass(t *testing.T) {
	if got := normalizeProducedInClass("Desc_SmelterMk1_C"); got != "Build_SmelterMk1_C" {
		t.Fatalf("smelter: %s", got)
	}
	if got := normalizeProducedInClass("BP_WorkBenchComponent_C"); got != "" {
		t.Fatalf("workbench excluded: %s", got)
	}
}

func TestDetectDataFormat(t *testing.T) {
	if got := DetectDataFormat("../../data/game-data.json"); got != "greeny" {
		t.Fatalf("game-data.json format: %q", got)
	}
	if got := DetectDataFormat("../../data/Docs.json"); got != "docs" {
		t.Fatalf("Docs.json format: %q", got)
	}
}
