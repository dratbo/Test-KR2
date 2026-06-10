package production

import (
	"sort"
	"strings"

	"github.com/dratbo/satisfactory-task-manager/gateway/internal/clients"
)

// MaterialRow is an aggregated resource needed for building construction.
type MaterialRow struct {
	ClassName   string
	DisplayName string
	IconURL     string
	Amount      float64
}

// BuildingCostRow lists resources for one building type.
type BuildingCostRow struct {
	BuildingName  string
	BuildingClass string
	Count         int
	Materials     []MaterialRow
}

// BuildClassToDesc converts Build_* class to Desc_* descriptor class.
func BuildClassToDesc(buildClass string) string {
	if strings.HasPrefix(buildClass, "Build_") {
		return "Desc_" + strings.TrimPrefix(buildClass, "Build_")
	}
	return buildClass
}

// IsBuildingRecipe reports whether a recipe is a handheld building recipe.
func IsBuildingRecipe(recipe *clients.Recipe) bool {
	if recipe == nil {
		return false
	}
	for _, p := range recipe.ProducedIn {
		if p == "BP_BuildGun_C" {
			return true
		}
	}
	return false
}

// AggregateBuildingCosts sums construction materials for all required buildings.
func AggregateBuildingCosts(
	rows []BuildingCostRow,
	itemName func(className string) string,
) []MaterialRow {
	totals := map[string]float64{}
	for _, row := range rows {
		for _, mat := range row.Materials {
			totals[mat.ClassName] += mat.Amount
		}
	}
	var out []MaterialRow
	for class, amount := range totals {
		out = append(out, MaterialRow{
			ClassName:   class,
			DisplayName: itemName(class),
			IconURL:     clients.ItemIconURL(class),
			Amount:      round2(amount),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].DisplayName < out[j].DisplayName
	})
	return out
}
