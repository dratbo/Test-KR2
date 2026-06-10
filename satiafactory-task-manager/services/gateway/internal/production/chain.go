package production

import (
	"strings"

	"github.com/dratbo/satisfactory-task-manager/gateway/internal/clients"
)

// PickChainRecipe chooses the best default recipe when expanding a production chain.
// Prefers standard machines over converters and skips zero-ingredient converter outputs.
func PickChainRecipe(recipes []clients.Recipe) *clients.Recipe {
	if len(recipes) == 0 {
		return nil
	}

	var filtered []clients.Recipe
	for _, r := range recipes {
		if len(r.Ingredients) == 0 && len(recipes) > 1 {
			continue
		}
		filtered = append(filtered, r)
	}
	if len(filtered) == 0 {
		filtered = recipes
	}

	var nonConverter []clients.Recipe
	for _, r := range filtered {
		if !isConverterRecipe(&r) {
			nonConverter = append(nonConverter, r)
		}
	}
	pool := filtered
	if len(nonConverter) > 0 {
		pool = nonConverter
	}

	best := &pool[0]
	bestPriority := recipePriority(best)
	for i := 1; i < len(pool); i++ {
		p := recipePriority(&pool[i])
		if p > bestPriority {
			best = &pool[i]
			bestPriority = p
		}
	}
	return best
}

func recipePriority(r *clients.Recipe) int {
	if r == nil {
		return 0
	}
	if r.ManufactoringMenuPriority > 0 {
		return r.ManufactoringMenuPriority
	}
	if isConverterRecipe(r) {
		return 3
	}
	return 10
}

func isConverterRecipe(r *clients.Recipe) bool {
	for _, p := range r.ProducedIn {
		if normalizeBuildingClass(p) == "Build_Converter_C" {
			return true
		}
	}
	return false
}

func normalizeBuildingClass(class string) string {
	if strings.HasPrefix(class, "Build_") {
		return class
	}
	if strings.HasPrefix(class, "Desc_") {
		return "Build_" + strings.TrimPrefix(class, "Desc_")
	}
	return class
}

// NeedsExtractor reports whether a chain leaf is mined or pumped from the world.
func NeedsExtractor(itemClass string, hasRecipe bool) bool {
	return IsExtractedResource(itemClass)
}

// IsUnplannedRawItem is a leaf without a crafting recipe that is not world-extracted
// (by-products like Plutonium Waste). These are omitted from factory building counts.
func IsUnplannedRawItem(itemClass string, hasRecipe bool) bool {
	return !hasRecipe && !IsExtractedResource(itemClass)
}
