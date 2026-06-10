package parser

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/dratbo/satisfactory-task-manager/satisfactory-data-service/internal/models"
	"github.com/dratbo/satisfactory-task-manager/satisfactory-data-service/internal/repository"
)

type greenyRoot struct {
	Items      map[string]greenyItem       `json:"items"`
	Recipes    map[string]greenyRecipe     `json:"recipes"`
	Miners     map[string]greenyMiner      `json:"miners"`
	Generators map[string]greenyGenerator  `json:"generators"`
	Buildings  map[string]greenyBuilding   `json:"buildings"`
	Schematics map[string]greenySchematic  `json:"schematics"`
}

type greenySchematic struct {
	ClassName string `json:"className"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Tier      int    `json:"tier"`
	Unlock    struct {
		Recipes []string `json:"recipes"`
	} `json:"unlock"`
}

type greenyItem struct {
	ClassName   string  `json:"className"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	StackSize   int     `json:"stackSize"`
	EnergyValue float64 `json:"energyValue"`
}

type greenyRecipe struct {
	ClassName   string           `json:"className"`
	Name        string           `json:"name"`
	Alternate   bool             `json:"alternate"`
	Time        float64          `json:"time"`
	InMachine   bool             `json:"inMachine"`
	ForBuilding bool             `json:"forBuilding"`
	Ingredients []greenyItemAmount `json:"ingredients"`
	Products    []greenyItemAmount `json:"products"`
	ProducedIn  []string         `json:"producedIn"`
}

type greenyItemAmount struct {
	Item   string  `json:"item"`
	Amount float64 `json:"amount"`
}

type greenyMiner struct {
	ClassName string `json:"className"`
}

type greenyGenerator struct {
	ClassName string `json:"className"`
}

type greenyBuilding struct {
	ClassName   string `json:"className"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// RunGreenyParser imports Satisfactory v1.0 data (SatisfactoryTools game-data.json format).
func RunGreenyParser(db *sql.DB, filePath string) error {
	log.Printf("Parsing game data (SatisfactoryTools v1.0 format): %s", filePath)
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var root greenyRoot
	if err := json.NewDecoder(file).Decode(&root); err != nil {
		return err
	}
	if len(root.Recipes) == 0 {
		return os.ErrInvalid
	}

	if err := ClearGameData(db); err != nil {
		return err
	}

	itemRepo := repository.NewItemRepository(db)
	buildingRepo := repository.NewBuildingRepository(db)
	recipeRepo := repository.NewRecipeRepository(db)
	schematicRepo := repository.NewSchematicRepository(db)
	unlockRepo := repository.NewUnlockRepository(db)

	itemCount := 0
	for _, item := range root.Items {
		if item.ClassName == "" || item.Name == "" {
			continue
		}
		if err := itemRepo.Insert(&models.Item{
			ClassName:   item.ClassName,
			DisplayName: item.Name,
			Description: item.Description,
			StackSize:   item.StackSize,
			EnergyValue: item.EnergyValue,
		}); err != nil {
			log.Printf("Warning: insert item %s: %v", item.ClassName, err)
		} else {
			itemCount++
		}
	}

	buildingNames := map[string]string{}
	for _, miner := range root.Miners {
		if miner.ClassName == "" {
			continue
		}
		buildClass := descToBuildClass(miner.ClassName)
		buildingNames[buildClass] = lookupGreenyName(root, miner.ClassName)
	}
	for _, gen := range root.Generators {
		if gen.ClassName == "" {
			continue
		}
		buildClass := descToBuildClass(gen.ClassName)
		buildingNames[buildClass] = lookupGreenyName(root, gen.ClassName)
	}

	recipeCount := 0
	for _, rec := range root.Recipes {
		if rec.ClassName == "" || rec.Name == "" {
			continue
		}
		for _, p := range rec.ProducedIn {
			buildClass := normalizeProducedInClass(p)
			if buildClass == "" {
				continue
			}
			if _, ok := buildingNames[buildClass]; !ok {
				buildingNames[buildClass] = lookupGreenyName(root, buildToDescClass(buildClass))
			}
		}

		ingredients := make([]models.Ingredient, 0, len(rec.Ingredients))
		for _, ing := range rec.Ingredients {
			if ing.Item == "" {
				continue
			}
			ingredients = append(ingredients, models.Ingredient{
				ItemClassName: ing.Item,
				Amount:        ing.Amount,
			})
		}
		products := make([]models.Product, 0, len(rec.Products))
		for _, prod := range rec.Products {
			if prod.Item == "" {
				continue
			}
			products = append(products, models.Product{
				ItemClassName: prod.Item,
				Amount:        prod.Amount,
			})
		}
		producedIn := make([]string, 0, len(rec.ProducedIn))
		for _, p := range rec.ProducedIn {
			if buildClass := normalizeProducedInClass(p); buildClass != "" {
				producedIn = append(producedIn, buildClass)
			}
		}

		priority := menuPriority(rec)
		if err := recipeRepo.Insert(&models.Recipe{
			ClassName:                 rec.ClassName,
			DisplayName:               rec.Name,
			Ingredients:               ingredients,
			Products:                  products,
			ProducedIn:                producedIn,
			Duration:                  rec.Time,
			ManufactoringMenuPriority: priority,
		}); err != nil {
			log.Printf("Warning: insert recipe %s: %v", rec.ClassName, err)
		} else {
			recipeCount++
		}
	}

	buildingCount := 0
	for class, name := range buildingNames {
		if name == "" {
			name = formatClassLabel(class)
		}
		if err := buildingRepo.Insert(&models.Building{
			ClassName:   class,
			DisplayName: name,
		}); err != nil {
			log.Printf("Warning: insert building %s: %v", class, err)
		} else {
			buildingCount++
		}
	}

	schematicCount := 0
	for _, sch := range root.Schematics {
		if sch.ClassName == "" {
			continue
		}
		hubTier := schematicHubTier(sch)
		if err := schematicRepo.Insert(&models.Schematic{
			ClassName:      sch.ClassName,
			DisplayName:    sch.Name,
			SchematicType:  sch.Type,
			HubTier:        hubTier,
		}); err != nil {
			log.Printf("Warning: insert schematic %s: %v", sch.ClassName, err)
			continue
		}
		schematicCount++
		for _, recipeClass := range sch.Unlock.Recipes {
			if recipeClass == "" {
				continue
			}
			if err := unlockRepo.UpsertRecipeTier(recipeClass, hubTier); err != nil {
				log.Printf("Warning: unlock tier %s: %v", recipeClass, err)
			}
			if rec, ok := root.Recipes[recipeClass]; ok {
				for _, prod := range rec.Products {
					if prod.Item == "" {
						continue
					}
					buildClass := descToBuildClass(prod.Item)
					if buildClass == "" {
						continue
					}
					if err := unlockRepo.UpsertBuildingTier(buildClass, hubTier); err != nil {
						log.Printf("Warning: building unlock %s: %v", buildClass, err)
					}
				}
			}
		}
	}

	log.Printf("Imported v1.0 data: %d items, %d buildings, %d recipes, %d schematics", itemCount, buildingCount, recipeCount, schematicCount)
	return nil
}

func schematicHubTier(sch greenySchematic) int {
	if sch.Type == "EST_Milestone" && sch.Tier > 0 {
		return sch.Tier
	}
	return 0
}

func menuPriority(rec greenyRecipe) int {
	if rec.InMachine {
		if rec.Alternate {
			return 5
		}
		return 10
	}
	if rec.ForBuilding {
		return 1
	}
	return 0
}

func lookupGreenyName(root greenyRoot, className string) string {
	if item, ok := root.Items[className]; ok && item.Name != "" {
		return item.Name
	}
	if b, ok := root.Buildings[className]; ok && b.Name != "" {
		return b.Name
	}
	for _, rec := range root.Recipes {
		for _, prod := range rec.Products {
			if prod.Item == className && rec.Name != "" {
				return rec.Name
			}
		}
	}
	return ""
}

func normalizeProducedInClass(className string) string {
	if className == "" || strings.HasPrefix(className, "BP_") {
		return ""
	}
	if strings.HasPrefix(className, "Build_") {
		return className
	}
	if strings.HasPrefix(className, "Desc_") {
		return descToBuildClass(className)
	}
	return ""
}

func descToBuildClass(desc string) string {
	if strings.HasPrefix(desc, "Desc_") {
		return "Build_" + strings.TrimPrefix(desc, "Desc_")
	}
	return desc
}

func buildToDescClass(build string) string {
	if strings.HasPrefix(build, "Build_") {
		return "Desc_" + strings.TrimPrefix(build, "Build_")
	}
	return build
}

func formatClassLabel(className string) string {
	s := strings.TrimSuffix(className, "_C")
	s = strings.TrimPrefix(s, "Build_")
	s = strings.ReplaceAll(s, "Mk1", "Mk.1")
	s = strings.ReplaceAll(s, "Mk2", "Mk.2")
	s = strings.ReplaceAll(s, "Mk3", "Mk.3")
	s = strings.ReplaceAll(s, "_", " ")
	return s
}

// DetectDataFormat returns "greeny", "docs", or "".
func DetectDataFormat(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()

	var peek struct {
		Items   json.RawMessage `json:"items"`
		Classes json.RawMessage `json:"Classes"`
	}
	if err := json.NewDecoder(file).Decode(&peek); err != nil {
		return ""
	}
	if len(peek.Items) > 0 {
		return "greeny"
	}
	if len(peek.Classes) > 0 {
		return "docs"
	}
	return ""
}

// RunImport picks the parser based on file format.
func RunImport(db *sql.DB, filePath string) error {
	switch DetectDataFormat(filePath) {
	case "greeny":
		return RunGreenyParser(db, filePath)
	case "docs":
		if err := ClearGameData(db); err != nil {
			return err
		}
		return RunParser(db, filePath)
	default:
		return os.ErrInvalid
	}
}
