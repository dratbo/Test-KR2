package production

import (
	"testing"

	"github.com/dratbo/satisfactory-task-manager/gateway/internal/clients"
)

func TestPickChainRecipePrefersConstructorOverConverter(t *testing.T) {
	recipes := []clients.Recipe{
		{
			ClassName:   "Recipe_DarkEnergy_C",
			DisplayName: "Dark Matter Residue",
			ProducedIn:  []string{"Desc_Converter_C"},
			Ingredients: []clients.Ingredient{{ItemClassName: "Desc_SAMIngot_C", Amount: 5}},
			Products:    []clients.Product{{ItemClassName: "Desc_DarkEnergy_C", Amount: 10}},
		},
		{
			ClassName:                 "Recipe_IngotSAM_C",
			DisplayName:               "Reanimated SAM",
			ManufactoringMenuPriority: 10,
			ProducedIn:                []string{"Desc_ConstructorMk1_C"},
			Ingredients:               []clients.Ingredient{{ItemClassName: "Desc_SAM_C", Amount: 4}},
			Products:                  []clients.Product{{ItemClassName: "Desc_SAMIngot_C", Amount: 1}},
		},
	}
	picked := PickChainRecipe(recipes)
	if picked == nil || picked.ClassName != "Recipe_IngotSAM_C" {
		t.Fatalf("expected constructor recipe, got %v", picked)
	}
}

func TestPickExtractorUsesMk3Miner(t *testing.T) {
	ext := PickExtractor("Desc_OreIron_C", nil)
	if ext.BuildingClass != "Build_MinerMk3_C" || ext.BaseRate != 240 {
		t.Fatalf("unexpected extractor: %+v", ext)
	}
}
