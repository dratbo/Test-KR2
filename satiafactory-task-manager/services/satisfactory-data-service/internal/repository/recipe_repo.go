package repository

import (
	"database/sql"
	"encoding/json"
	"github.com/dratbo/satisfactory-task-manager/satisfactory-data-service/internal/models"
)

type RecipeRepository struct {
	db *sql.DB
}

func NewRecipeRepository(db *sql.DB) *RecipeRepository {
	return &RecipeRepository{db: db}
}

func (r *RecipeRepository) Insert(recipe *models.Recipe) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	producedInJSON, _ := json.Marshal(recipe.ProducedIn)
	query := `INSERT INTO recipes (class_name, display_name, produced_in, duration, manufactoring_menu_priority)
	          VALUES ($1, $2, $3, $4, $5) ON CONFLICT (class_name) DO UPDATE SET
	              display_name = EXCLUDED.display_name,
	              produced_in = EXCLUDED.produced_in,
	              duration = EXCLUDED.duration,
	              manufactoring_menu_priority = EXCLUDED.manufactoring_menu_priority`
	_, err = tx.Exec(query, recipe.ClassName, recipe.DisplayName, producedInJSON, recipe.Duration, recipe.ManufactoringMenuPriority)
	if err != nil {
		return err
	}

	// Вставка ингредиентов
	for _, ing := range recipe.Ingredients {
		_, err = tx.Exec(`INSERT INTO recipe_ingredients (recipe_class_name, item_class_name, amount) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
			recipe.ClassName, ing.ItemClassName, ing.Amount)
		if err != nil {
			return err
		}
	}
	// Вставка продуктов
	for _, prod := range recipe.Products {
		_, err = tx.Exec(`INSERT INTO recipe_products (recipe_class_name, item_class_name, amount) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
			recipe.ClassName, prod.ItemClassName, prod.Amount)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *RecipeRepository) GetAll() ([]models.Recipe, error) {
	rows, err := r.db.Query(`SELECT class_name, display_name, produced_in, duration, manufactoring_menu_priority FROM recipes ORDER BY display_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recipes []models.Recipe
	for rows.Next() {
		var rec models.Recipe
		var producedInJSON []byte
		err := rows.Scan(&rec.ClassName, &rec.DisplayName, &producedInJSON, &rec.Duration, &rec.ManufactoringMenuPriority)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(producedInJSON, &rec.ProducedIn)
		// Загружаем ингредиенты
		ingRows, err := r.db.Query(`SELECT item_class_name, amount FROM recipe_ingredients WHERE recipe_class_name = $1`, rec.ClassName)
		if err != nil {
			return nil, err
		}
		for ingRows.Next() {
			var ing models.Ingredient
			if err := ingRows.Scan(&ing.ItemClassName, &ing.Amount); err != nil {
				ingRows.Close()
				return nil, err
			}
			rec.Ingredients = append(rec.Ingredients, ing)
		}
		ingRows.Close()
		// Загружаем продукты
		prodRows, err := r.db.Query(`SELECT item_class_name, amount FROM recipe_products WHERE recipe_class_name = $1`, rec.ClassName)
		if err != nil {
			return nil, err
		}
		for prodRows.Next() {
			var prod models.Product
			if err := prodRows.Scan(&prod.ItemClassName, &prod.Amount); err != nil {
				prodRows.Close()
				return nil, err
			}
			rec.Products = append(rec.Products, prod)
		}
		prodRows.Close()
		recipes = append(recipes, rec)
	}
	return recipes, nil
}
