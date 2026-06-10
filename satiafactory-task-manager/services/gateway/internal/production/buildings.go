package production

import "strings"

// Buildings excluded from automated production planning.
var excludedBuildings = map[string]bool{
	"BP_WorkBenchComponent_C":           true,
	"FGBuildableAutomatedWorkBench":     true,
	"FGBuildableAutomatedWorkBench_C":   true,
}

// buildingNamesRU maps factory building class names to Russian display names.
var buildingNamesRU = map[string]string{
	"Build_ConstructorMk1_C":   "Конструктор",
	"Build_SmelterMk1_C":       "Плавильня",
	"Build_FoundryMk1_C":       "Литейная",
	"Build_AssemblerMk1_C":     "Сборочный цех",
	"Build_ManufacturerMk1_C":  "Производитель",
	"Build_Blender_C":          "Смеситель",
	"Build_Packager_C":         "Упаковщик",
	"Build_OilPump_C":          "Нефтяная вышка",
	"Build_WaterExtractor_C":   "Водяной насос",
	"Build_MinerMk1_C":         "Буровая установка Mk.1",
	"Build_MinerMk2_C":         "Буровая установка Mk.2",
	"Build_MinerMk3_C":         "Буровая установка Mk.3",
	"Build_Refinery_C":         "Нефтеперерабатывающий завод",
	"Build_GeneratorBiomass_C": "Биомассовый генератор",
	"Build_GeneratorCoal_C":    "Угольный генератор",
}

// extractorBaseRate is the normal node extraction rate (items/min at 100% clock).
const extractorBaseRate = 60.0

// PickFactoryBuilding returns the primary automated factory building from produced_in.
func PickFactoryBuilding(producedIn []string) string {
	for _, class := range producedIn {
		if excludedBuildings[class] {
			continue
		}
		if strings.HasPrefix(class, "Build_") {
			return class
		}
	}
	return ""
}

// SupportsPowerShards reports whether the building can use power modules (0–3).
func SupportsPowerShards(buildingClass string) bool {
	if buildingClass == "" {
		return false
	}
	if strings.HasPrefix(buildingClass, "Build_") && !excludedBuildings[buildingClass] {
		return true
	}
	return false
}

// BuildingDisplayName returns a human-readable building name (Russian when known).
func BuildingDisplayName(className, englishName string) string {
	if ru, ok := buildingNamesRU[className]; ok {
		return ru
	}
	if englishName != "" {
		return englishName
	}
	return formatClassName(className)
}

func formatClassName(className string) string {
	s := strings.TrimSuffix(className, "_C")
	s = strings.TrimPrefix(s, "Build_")
	s = strings.ReplaceAll(s, "Mk1", "Mk.1")
	s = strings.ReplaceAll(s, "Mk2", "Mk.2")
	s = strings.ReplaceAll(s, "Mk3", "Mk.3")
	s = strings.ReplaceAll(s, "_", " ")
	return s
}

// DefaultMinerClass is used for raw resources extracted from resource nodes.
const DefaultMinerClass = "Build_MinerMk1_C"
