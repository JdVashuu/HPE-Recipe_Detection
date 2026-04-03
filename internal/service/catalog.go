package service

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/JdVashuu/RecipeDetection.git/internal/model"
)

type CatalogService struct {
	catalogs []model.Catalog
}

func NewCatalogService() *CatalogService {
	return &CatalogService{
		catalogs: buildCatalogs("../data/recipes_hpe.json"),
	}
}

func (s *CatalogService) GetAllCatalogs() []model.Catalog {
	return s.catalogs
}

func (s *CatalogService) GetRecipesByCatalog(recipeVersion string) model.Recipe {
	for _, c := range s.catalogs {
		for _, r := range c.Recipes {
			if r.Version == recipeVersion {
				return r
			}
		}
	}
	return model.Recipe{}
}

func (s *CatalogService) GetComponentByRecipe(recipeVersion string) map[string]model.Component {
	for _, c := range s.catalogs {
		for _, r := range c.Recipes {
			if r.Version == recipeVersion {
				return r.UpdateComponents.Components
			}
		}
	}
	return map[string]model.Component{}
}

func buildCatalogs(path string) []model.Catalog {
	file, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Failed to open the recipe JSON %v : %v\n", path, err)
		return nil
	}

	var catalog model.Catalog
	if err := json.Unmarshal(file, &catalog); err != nil {
		fmt.Printf("Failed to unmarshall JSON: %v\n", err)
		return nil
	}

	return []model.Catalog{catalog}
}

func (s *CatalogService) GetUpgradePaths(recipeVersion string) []string {
	for _, c := range s.catalogs {
		for _, r := range c.Recipes {
			if r.Version == recipeVersion {
				return r.UpgradeTo
			}
		}
	}
	return []string{}
}
