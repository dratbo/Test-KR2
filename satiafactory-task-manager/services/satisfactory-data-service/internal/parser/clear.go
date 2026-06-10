package parser

import "database/sql"

// ClearGameData removes imported game data before a full re-import.
func ClearGameData(db *sql.DB) error {
	stmts := []string{
		`TRUNCATE schematic_unlocks, schematic_costs, recipe_unlock_tiers, building_unlock_tiers, schematics, recipe_ingredients, recipe_products, recipes, buildings, items RESTART IDENTITY CASCADE`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
