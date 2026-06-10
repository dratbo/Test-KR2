package production

import "testing"

func TestIsExtractedResource(t *testing.T) {
	raw := []string{
		"Desc_OreIron_C",
		"Desc_Coal_C",
		"Desc_Sulfur_C",
		"Desc_Stone_C",
		"Desc_SAM_C",
		"Desc_Water_C",
	}
	for _, class := range raw {
		if !IsExtractedResource(class) {
			t.Fatalf("expected %s to be extracted resource", class)
		}
	}
	crafted := []string{
		"Desc_IronIngot_C",
		"Desc_IronPlate_C",
		"Desc_SAMIngot_C",
		"Desc_Limestone_C",
	}
	for _, class := range crafted {
		if IsExtractedResource(class) {
			t.Fatalf("expected %s to be crafted, not extracted", class)
		}
	}
}
