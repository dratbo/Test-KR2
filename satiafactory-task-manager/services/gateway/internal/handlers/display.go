package handlers

import (
	"strings"

	"github.com/dratbo/satisfactory-task-manager/gateway/internal/clients"
	"github.com/dratbo/satisfactory-task-manager/gateway/internal/production"
)

func formatItemLocalized(item *clients.Item, className string) string {
	en := ""
	ru := ""
	if item != nil {
		en = strings.TrimSpace(item.DisplayName)
		ru = strings.TrimSpace(item.DisplayNameRU)
	}
	if en == "" {
		en = formatItemName(className)
	}
	if ru != "" && en != "" && !strings.EqualFold(ru, en) {
		return ru + " (" + en + ")"
	}
	if ru != "" {
		return ru
	}
	return en
}

func itemDisplayName(dataClient *clients.DataClient, className string) string {
	item, _ := dataClient.GetItem(className)
	return formatItemLocalized(item, className)
}

func ingredientCraftable(dataClient *clients.DataClient, itemClass string) bool {
	if production.IsExtractedResource(itemClass) {
		return false
	}
	ok, _ := dataClient.HasRecipeForProduct(itemClass)
	return ok
}

func recipeDisplayTitle(recipe *clients.Recipe) string {
	if recipe == nil {
		return ""
	}
	en := strings.TrimSpace(recipe.DisplayName)
	ru := strings.TrimSpace(recipe.DisplayNameRU)
	if ru != "" && en != "" && !strings.EqualFold(ru, en) {
		return ru + " (" + en + ")"
	}
	if ru != "" {
		return ru
	}
	return en
}

func recipeProductPerCycle(recipe *clients.Recipe) float64 {
	if recipe == nil || len(recipe.Products) == 0 {
		return 1
	}
	return recipe.Products[0].Amount
}

func ingredientRatePerMin(recipe *clients.Recipe, targetPerMin, ingPerCycle float64) float64 {
	prodPerCycle := recipeProductPerCycle(recipe)
	if prodPerCycle <= 0 || targetPerMin <= 0 {
		return 0
	}
	return round2(targetPerMin * ingPerCycle / prodPerCycle)
}

func productRatePerMin(recipe *clients.Recipe, targetPerMin, prodPerCycle float64) float64 {
	base := recipeProductPerCycle(recipe)
	if base <= 0 || targetPerMin <= 0 {
		return 0
	}
	return round2(targetPerMin * prodPerCycle / base)
}
