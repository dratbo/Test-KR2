package clients

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type DataClient struct {
	baseURL string
	client  *http.Client
}

func NewDataClient(baseURL string) *DataClient {
	if baseURL == "" {
		baseURL = os.Getenv("DATA_SERVICE_URL")
	}
	if baseURL == "" {
		baseURL = "http://localhost:8083"
	}
	return &DataClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

type Ingredient struct {
	ItemClassName string  `json:"item_class_name"`
	Amount        float64 `json:"amount"`
}

type Product struct {
	ItemClassName string  `json:"item_class_name"`
	Amount        float64 `json:"amount"`
}

type Item struct {
	ClassName     string `json:"class_name"`
	DisplayName   string `json:"display_name"`
	DisplayNameRU string `json:"display_name_ru,omitempty"`
}

type Recipe struct {
	ClassName                 string       `json:"class_name"`
	DisplayName               string       `json:"display_name"`
	DisplayNameRU             string       `json:"display_name_ru,omitempty"`
	Ingredients               []Ingredient `json:"ingredients"`
	Products                  []Product    `json:"products"`
	ProducedIn                []string     `json:"produced_in"`
	Duration                  float64      `json:"duration"`
	ManufactoringMenuPriority int          `json:"manufactoring_menu_priority"`
}

type Building struct {
	ClassName   string  `json:"class_name"`
	DisplayName string  `json:"display_name"`
	Description string  `json:"description"`
	PowerConsumption float64 `json:"power_consumption"`
}

type UnlockIndex struct {
	RecipeTiers   map[string]int `json:"recipe_tiers"`
	BuildingTiers map[string]int `json:"building_tiers"`
}

func (c *DataClient) SearchRecipes(query string, includeAlternates bool) ([]Recipe, error) {
	u := c.baseURL + "/api/recipes/search?q=" + url.QueryEscape(query)
	if includeAlternates {
		u += "&include_alternates=1"
	}
	resp, err := c.client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search recipes: %s", resp.Status)
	}
	var recipes []Recipe
	if err := json.NewDecoder(resp.Body).Decode(&recipes); err != nil {
		return nil, err
	}
	return recipes, nil
}

func (c *DataClient) GetRecipesByProduct(itemClass string, includeAlternates bool) ([]Recipe, error) {
	u := c.baseURL + "/api/recipes/by-product/" + url.PathEscape(itemClass)
	if includeAlternates {
		u += "?include_alternates=1"
	}
	resp, err := c.client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("recipes by product: %s", resp.Status)
	}
	var recipes []Recipe
	if err := json.NewDecoder(resp.Body).Decode(&recipes); err != nil {
		return nil, err
	}
	return recipes, nil
}

func (c *DataClient) HasRecipeForProduct(itemClass string) (bool, error) {
	u := c.baseURL + "/api/recipes/has-product/" + url.PathEscape(itemClass)
	resp, err := c.client.Get(u)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("has recipe: %s", resp.Status)
	}
	var out struct {
		Craftable bool `json:"craftable"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, err
	}
	return out.Craftable, nil
}

func (c *DataClient) GetRecipe(className string) (*Recipe, error) {
	u := c.baseURL + "/api/recipes/" + url.PathEscape(className)
	resp, err := c.client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get recipe: %s", resp.Status)
	}
	var recipe Recipe
	if err := json.NewDecoder(resp.Body).Decode(&recipe); err != nil {
		return nil, err
	}
	return &recipe, nil
}

func buildClassToDesc(buildClass string) string {
	if strings.HasPrefix(buildClass, "Build_") {
		return "Desc_" + strings.TrimPrefix(buildClass, "Build_")
	}
	return buildClass
}

func (c *DataClient) GetBuildingRecipe(buildClass string) (*Recipe, error) {
	descClass := buildClassToDesc(buildClass)
	recipes, err := c.GetRecipesByProduct(descClass, false)
	if err != nil || len(recipes) == 0 {
		return nil, err
	}
	for i := range recipes {
		for _, p := range recipes[i].ProducedIn {
			if p == "BP_BuildGun_C" {
				return &recipes[i], nil
			}
		}
	}
	return &recipes[0], nil
}

func (c *DataClient) GetBuildings() ([]Building, error) {
	resp, err := c.client.Get(c.baseURL + "/api/buildings")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get buildings: %s", resp.Status)
	}
	var buildings []Building
	if err := json.NewDecoder(resp.Body).Decode(&buildings); err != nil {
		return nil, err
	}
	return buildings, nil
}

func (c *DataClient) GetUnlockIndex() (*UnlockIndex, error) {
	resp, err := c.client.Get(c.baseURL + "/api/unlocks")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get unlocks: %s", resp.Status)
	}
	var idx UnlockIndex
	if err := json.NewDecoder(resp.Body).Decode(&idx); err != nil {
		return nil, err
	}
	if idx.RecipeTiers == nil {
		idx.RecipeTiers = map[string]int{}
	}
	if idx.BuildingTiers == nil {
		idx.BuildingTiers = map[string]int{}
	}
	return &idx, nil
}

func (c *DataClient) GetItem(className string) (*Item, error) {
	u := c.baseURL + "/api/items/" + url.PathEscape(className)
	resp, err := c.client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get item: %s", resp.Status)
	}
	var item Item
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, err
	}
	return &item, nil
}

// ItemIconURL returns a gateway-proxied icon URL for the given item class name.
func ItemIconURL(className string) string {
	if className == "" {
		return "/static/placeholder.svg"
	}
	return "/icons/" + url.PathEscape(className) + ".png"
}
