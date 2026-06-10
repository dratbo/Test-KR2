package production

import "strings"

// extractedResources are world nodes (miners / pumps / geysers), not craft outputs.
// v1.0 converter recipes can list these as products; the production chain must stop here.
var extractedResources = map[string]struct{}{
	"Desc_OreIron_C":     {},
	"Desc_OreCopper_C":   {},
	"Desc_OreGold_C":     {},
	"Desc_OreBauxite_C":  {},
	"Desc_OreUranium_C":  {},
	"Desc_RawQuartz_C":   {},
	"Desc_Coal_C":        {},
	"Desc_Sulfur_C":      {},
	"Desc_Stone_C":       {}, // Limestone
	"Desc_SAM_C":         {},
	"Desc_LiquidOil_C":   {},
	"Desc_Water_C":       {},
	"Desc_NitrogenGas_C": {},
}

func IsFluidItem(itemClass string) bool {
	switch itemClass {
	case "Desc_Water_C", "Desc_LiquidOil_C", "Desc_NitrogenGas_C",
		"Desc_LiquidFuel_C", "Desc_LiquidTurboFuel_C", "Desc_LiquidBiofuel_C",
		"Desc_AluminaSolution_C", "Desc_SulfuricAcid_C", "Desc_NitricAcid_C",
		"Desc_HeavyOilResidue_C":
		return true
	default:
		return strings.HasPrefix(itemClass, "Desc_Liquid")
	}
}

func IsExtractedResource(itemClass string) bool {
	if itemClass == "" {
		return false
	}
	_, ok := extractedResources[itemClass]
	return ok
}
