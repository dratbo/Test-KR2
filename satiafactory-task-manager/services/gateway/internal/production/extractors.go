package production

// Extractor describes the best available resource extractor for planning.
type Extractor struct {
	BuildingClass string
	BaseRate      float64 // items/min at 100% clock, normal node
}

// PickExtractor returns the best unlocked extractor for a world resource.
func PickExtractor(itemClass string, tier *TierContext) Extractor {
	return pickBestExtractor(itemClass, tier)
}
