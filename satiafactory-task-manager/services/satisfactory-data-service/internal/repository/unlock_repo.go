package repository

import (
	"database/sql"
)

type UnlockRepository struct {
	db *sql.DB
}

func NewUnlockRepository(db *sql.DB) *UnlockRepository {
	return &UnlockRepository{db: db}
}

func (r *UnlockRepository) Clear() error {
	_, err := r.db.Exec(`TRUNCATE recipe_unlock_tiers, building_unlock_tiers RESTART IDENTITY`)
	return err
}

func (r *UnlockRepository) UpsertRecipeTier(recipeClass string, hubTier int) error {
	_, err := r.db.Exec(
		`INSERT INTO recipe_unlock_tiers (recipe_class_name, hub_tier) VALUES ($1, $2)
		 ON CONFLICT (recipe_class_name) DO UPDATE SET hub_tier = LEAST(recipe_unlock_tiers.hub_tier, EXCLUDED.hub_tier)`,
		recipeClass, hubTier,
	)
	return err
}

func (r *UnlockRepository) UpsertBuildingTier(buildingClass string, hubTier int) error {
	_, err := r.db.Exec(
		`INSERT INTO building_unlock_tiers (building_class_name, hub_tier) VALUES ($1, $2)
		 ON CONFLICT (building_class_name) DO UPDATE SET hub_tier = LEAST(building_unlock_tiers.hub_tier, EXCLUDED.hub_tier)`,
		buildingClass, hubTier,
	)
	return err
}

type UnlockIndex struct {
	RecipeTiers   map[string]int `json:"recipe_tiers"`
	BuildingTiers map[string]int `json:"building_tiers"`
}

func (r *UnlockRepository) GetIndex() (*UnlockIndex, error) {
	idx := &UnlockIndex{
		RecipeTiers:   map[string]int{},
		BuildingTiers: map[string]int{},
	}

	rows, err := r.db.Query(`SELECT recipe_class_name, hub_tier FROM recipe_unlock_tiers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var class string
		var tier int
		if err := rows.Scan(&class, &tier); err != nil {
			return nil, err
		}
		idx.RecipeTiers[class] = tier
	}

	rows2, err := r.db.Query(`SELECT building_class_name, hub_tier FROM building_unlock_tiers`)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var class string
		var tier int
		if err := rows2.Scan(&class, &tier); err != nil {
			return nil, err
		}
		idx.BuildingTiers[class] = tier
	}
	return idx, nil
}
