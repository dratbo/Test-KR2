package repository

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/dratbo/satisfactory-task-manager/satisfactory-data-service/internal/i18n"
	"github.com/dratbo/satisfactory-task-manager/satisfactory-data-service/internal/models"
)

func ensureItemStub(tx *sql.Tx, className string) error {
	if className == "" {
		return nil
	}
	display := strings.TrimSuffix(className, "_C")
	display = strings.TrimPrefix(display, "Desc_")
	display = strings.ReplaceAll(display, "_", " ")
	_, err := tx.Exec(
		`INSERT INTO items (class_name, display_name) VALUES ($1, $2) ON CONFLICT (class_name) DO NOTHING`,
		className, display,
	)
	return err
}

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

	for _, ing := range recipe.Ingredients {
		if err := ensureItemStub(tx, ing.ItemClassName); err != nil {
			return err
		}
	}
	for _, prod := range recipe.Products {
		if err := ensureItemStub(tx, prod.ItemClassName); err != nil {
			return err
		}
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

func (r *RecipeRepository) loadRecipeDetails(rec *models.Recipe) error {
	ingRows, err := r.db.Query(`SELECT item_class_name, amount FROM recipe_ingredients WHERE recipe_class_name = $1`, rec.ClassName)
	if err != nil {
		return err
	}
	defer ingRows.Close()
	for ingRows.Next() {
		var ing models.Ingredient
		if err := ingRows.Scan(&ing.ItemClassName, &ing.Amount); err != nil {
			return err
		}
		rec.Ingredients = append(rec.Ingredients, ing)
	}

	prodRows, err := r.db.Query(`SELECT item_class_name, amount FROM recipe_products WHERE recipe_class_name = $1`, rec.ClassName)
	if err != nil {
		return err
	}
	defer prodRows.Close()
	for prodRows.Next() {
		var prod models.Product
		if err := prodRows.Scan(&prod.ItemClassName, &prod.Amount); err != nil {
			return err
		}
		rec.Products = append(rec.Products, prod)
	}
	return nil
}

func (r *RecipeRepository) GetByClassName(className string) (*models.Recipe, error) {
	var rec models.Recipe
	var producedInJSON []byte
	err := r.db.QueryRow(
		`SELECT class_name, display_name, produced_in, duration, manufactoring_menu_priority FROM recipes WHERE class_name = $1`,
		className,
	).Scan(&rec.ClassName, &rec.DisplayName, &producedInJSON, &rec.Duration, &rec.ManufactoringMenuPriority)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(producedInJSON, &rec.ProducedIn)
	if err := r.loadRecipeDetails(&rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *RecipeRepository) Search(query string, limit int, includeAlternates bool) ([]models.Recipe, error) {
	if limit <= 0 {
		limit = 40
	}
	terms := i18n.SearchTerms(query)
	if len(terms) == 0 {
		return nil, nil
	}

	seen := map[string]struct{}{}
	var recipes []models.Recipe

	for _, term := range terms {
		if len(recipes) >= limit {
			break
		}
		batch, err := r.searchTerm(term, limit-len(recipes), includeAlternates)
		if err != nil {
			return nil, err
		}
		for _, rec := range batch {
			if _, ok := seen[rec.ClassName]; ok {
				continue
			}
			seen[rec.ClassName] = struct{}{}
			recipes = append(recipes, rec)
		}
	}
	return recipes, nil
}

func (r *RecipeRepository) searchTerm(term string, limit int, includeAlternates bool) ([]models.Recipe, error) {
	altFilter := ` AND class_name NOT LIKE '%Alternate%' AND display_name NOT LIKE 'Alternate:%' `
	if includeAlternates {
		altFilter = ""
	}

	rows, err := r.db.Query(
		`SELECT class_name, display_name, produced_in, duration, manufactoring_menu_priority
		 FROM recipes
		 WHERE display_name ILIKE '%' || $1 || '%'`+altFilter+`
		 ORDER BY display_name
		 LIMIT $2`,
		term, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipes []models.Recipe
	for rows.Next() {
		var rec models.Recipe
		var producedInJSON []byte
		if err := rows.Scan(&rec.ClassName, &rec.DisplayName, &producedInJSON, &rec.Duration, &rec.ManufactoringMenuPriority); err != nil {
			return nil, err
		}
		if !includeAlternates && i18n.IsAlternateRecipe(rec.ClassName, rec.DisplayName) {
			continue
		}
		_ = json.Unmarshal(producedInJSON, &rec.ProducedIn)
		if err := r.loadRecipeDetails(&rec); err != nil {
			return nil, err
		}
		fillDisplayNameRU(&rec)
		recipes = append(recipes, rec)
	}

	// Match by Russian localized product names
	for class, ru := range matchRuNames(term) {
		if len(recipes) >= limit {
			break
		}
		rec, err := r.findRecipeByProduct(class, includeAlternates)
		if err != nil || rec == nil {
			continue
		}
		if recipeSeen(recipes, rec.ClassName) {
			continue
		}
		rec.DisplayNameRU = ru
		recipes = append(recipes, *rec)
	}

	// Match by English product/item display names
	if len(recipes) < limit {
		byItem, err := r.searchByItemDisplayName(term, limit-len(recipes), includeAlternates)
		if err == nil {
			for _, rec := range byItem {
				if recipeSeen(recipes, rec.ClassName) {
					continue
				}
				fillDisplayNameRU(&rec)
				recipes = append(recipes, rec)
				if len(recipes) >= limit {
					break
				}
			}
		}
	}

	return recipes, nil
}

func recipeSeen(recipes []models.Recipe, className string) bool {
	for _, existing := range recipes {
		if existing.ClassName == className {
			return true
		}
	}
	return false
}

func (r *RecipeRepository) searchByItemDisplayName(term string, limit int, includeAlternates bool) ([]models.Recipe, error) {
	altFilter := ` AND r.class_name NOT LIKE '%Alternate%' AND r.display_name NOT LIKE 'Alternate:%' `
	if includeAlternates {
		altFilter = ""
	}
	rows, err := r.db.Query(
		`SELECT DISTINCT r.class_name, r.display_name, r.produced_in, r.duration, r.manufactoring_menu_priority
		 FROM recipes r
		 JOIN recipe_products p ON p.recipe_class_name = r.class_name
		 JOIN items i ON i.class_name = p.item_class_name
		 WHERE i.display_name ILIKE '%' || $1 || '%'`+altFilter+`
		 ORDER BY r.display_name
		 LIMIT $2`,
		term, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipes []models.Recipe
	for rows.Next() {
		var rec models.Recipe
		var producedInJSON []byte
		if err := rows.Scan(&rec.ClassName, &rec.DisplayName, &producedInJSON, &rec.Duration, &rec.ManufactoringMenuPriority); err != nil {
			return nil, err
		}
		if !includeAlternates && i18n.IsAlternateRecipe(rec.ClassName, rec.DisplayName) {
			continue
		}
		_ = json.Unmarshal(producedInJSON, &rec.ProducedIn)
		if err := r.loadRecipeDetails(&rec); err != nil {
			return nil, err
		}
		recipes = append(recipes, rec)
	}
	return recipes, nil
}

func fillDisplayNameRU(rec *models.Recipe) {
	rec.DisplayNameRU = i18n.NameRU(rec.ClassName)
	if rec.DisplayNameRU == "" && len(rec.Products) > 0 {
		rec.DisplayNameRU = i18n.NameRU(rec.Products[0].ItemClassName)
	}
}

func matchRuNames(term string) map[string]string {
	out := map[string]string{}
	lower := strings.ToLower(term)
	for class, ru := range i18n.AllNames() {
		if strings.Contains(strings.ToLower(ru), lower) {
			out[class] = ru
		}
	}
	return out
}

func (r *RecipeRepository) GetByProduct(itemClass string, includeAlternates bool) ([]models.Recipe, error) {
	altFilter := ` AND r.class_name NOT LIKE '%Alternate%' AND r.display_name NOT LIKE 'Alternate:%' `
	if includeAlternates {
		altFilter = ""
	}
	rows, err := r.db.Query(
		`SELECT r.class_name, r.display_name, r.produced_in, r.duration, r.manufactoring_menu_priority
		 FROM recipes r
		 JOIN recipe_products p ON p.recipe_class_name = r.class_name
		 WHERE p.item_class_name = $1`+altFilter+`
		 ORDER BY
		   CASE WHEN r.class_name LIKE '%Alternate%' OR r.display_name LIKE 'Alternate:%' THEN 1 ELSE 0 END,
		   r.display_name`,
		itemClass,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipes []models.Recipe
	for rows.Next() {
		var rec models.Recipe
		var producedInJSON []byte
		if err := rows.Scan(&rec.ClassName, &rec.DisplayName, &producedInJSON, &rec.Duration, &rec.ManufactoringMenuPriority); err != nil {
			return nil, err
		}
		if !includeAlternates && i18n.IsAlternateRecipe(rec.ClassName, rec.DisplayName) {
			continue
		}
		_ = json.Unmarshal(producedInJSON, &rec.ProducedIn)
		if err := r.loadRecipeDetails(&rec); err != nil {
			return nil, err
		}
		fillDisplayNameRU(&rec)
		if rec.DisplayNameRU == "" {
			rec.DisplayNameRU = i18n.NameRU(itemClass)
		}
		recipes = append(recipes, rec)
	}
	return recipes, nil
}

func (r *RecipeRepository) HasRecipeForProduct(itemClass string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM recipe_products WHERE item_class_name = $1)`,
		itemClass,
	).Scan(&exists)
	return exists, err
}

func (r *RecipeRepository) findRecipeByProduct(itemClass string, includeAlternates bool) (*models.Recipe, error) {
	recipes, err := r.GetByProduct(itemClass, includeAlternates)
	if err != nil {
		return nil, err
	}
	if len(recipes) == 0 {
		return nil, sql.ErrNoRows
	}
	return &recipes[0], nil
}

func (r *RecipeRepository) Count() (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM recipes`).Scan(&n)
	return n, err
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
		_ = json.Unmarshal(producedInJSON, &rec.ProducedIn)
		if err := r.loadRecipeDetails(&rec); err != nil {
			return nil, err
		}
		recipes = append(recipes, rec)
	}
	return recipes, nil
}
