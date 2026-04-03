package handler

import (
	"encoding/json"
	"net/http"

	"github.com/JdVashuu/RecipeDetection.git/internal/service"
	"github.com/go-chi/chi/v5"
)

type CatalogHandler struct {
	svc *service.CatalogService
}

func NewCatalogHandler(svc *service.CatalogService) *CatalogHandler {
	return &CatalogHandler{
		svc: svc,
	}
}

func (h *CatalogHandler) GetAllCatalogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.svc.GetAllCatalogs())
}

func (h *CatalogHandler) GetRecipeByCatalogs(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "catalogVersion")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.svc.GetRecipesByCatalog(version))
}

func (h *CatalogHandler) GetComponentByRecipe(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "recipeVersion")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.svc.GetComponentByRecipe(version))

}

func (h *CatalogHandler) GetUpgradePathsByRecipe(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "recipeVersion")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.svc.GetUpgradePaths(version))
}
