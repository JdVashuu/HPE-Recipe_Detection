package handler

import (
	"encoding/json"
	"net/http"

	"github.com/JdVashuu/RecipeDetection.git/internal/model"
	"github.com/JdVashuu/RecipeDetection.git/internal/service"
	"github.com/JdVashuu/RecipeDetection.git/internal/websocket"
	"github.com/go-chi/chi/v5"
	gorilla "github.com/gorilla/websocket"
)

type HelmReleaseHandler struct {
	svc    *service.HelmReleaseService
	gitops *service.GitOpsService
	hub    *websocket.Hub
}

func NewHelmReleaseHandler(svc *service.HelmReleaseService, gitops *service.GitOpsService, hub *websocket.Hub) *HelmReleaseHandler {
	return &HelmReleaseHandler{
		svc:    svc,
		gitops: gitops,
		hub:    hub,
	}
}

func (h *HelmReleaseHandler) GetAllHelmReleases(w http.ResponseWriter, r *http.Request) {
	releases, err := h.svc.GetAllHelmRelease(r.Context())
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	lightweight := []map[string]interface{}{}
	for _, r := range releases {
		lightweight = append(lightweight, map[string]interface{}{
			"version":     r.Version,
			"releaseName": r.ReleaseName,
			"status":      r.Status,
		})
	}
	h.respondWithJSON(w, http.StatusOK, lightweight)
}

func (h *HelmReleaseHandler) GetHelmReleases(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	release, err := h.svc.GetHelmRelease(r.Context(), version)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, err.Error())
		return
	}
	h.respondWithJSON(w, http.StatusOK, release)
}

func (h *HelmReleaseHandler) CreateHelmRelease(w http.ResponseWriter, r *http.Request) {
	var release model.HelmRelease
	if err := json.NewDecoder(r.Body).Decode(&release); err != nil {
		h.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	created, err := h.svc.CreateHelmRelease(r.Context(), &release)
	if err != nil {
		h.respondWithError(w, http.StatusConflict, err.Error())
		return
	}

	h.hub.Broadcast("release_created", created)
	h.respondWithJSON(w, http.StatusCreated, created)
}

func (h *HelmReleaseHandler) UpdateHelmRelease(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	var release model.HelmRelease
	if err := json.NewDecoder(r.Body).Decode(&release); err != nil {
		h.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := h.svc.UpdateHelmRelease(r.Context(), version, &release)
	if err != nil {
		h.respondWithError(w, http.StatusConflict, err.Error())
		return
	}

	h.hub.Broadcast("release_updated", updated)
	h.respondWithJSON(w, http.StatusOK, updated)

}

func (h *HelmReleaseHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	var body struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.respondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	release, err := h.svc.GetHelmRelease(r.Context(), version)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	release.Status = body.Status
	_, err = h.svc.UpdateHelmRelease(r.Context(), version, release)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.hub.Broadcast("status_changed", map[string]interface{}{"version": version, "status": body.Status})
	h.respondWithJSON(w, http.StatusOK, release)
}

func (h *HelmReleaseHandler) DeployRelease(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	release, err := h.svc.GetHelmRelease(r.Context(), version)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	if len(release.Recipes) == 0 {
		h.respondWithError(w, http.StatusBadRequest, "Cannot deploy a release with no recipes")
		return
	}

	release.Status = "deploying"
	h.hub.Broadcast("status_changed", map[string]interface{}{"version": version, "status": "deploying"})

	go func() {
		err := h.gitops.GenerateAndPush(release)
		if err != nil {
			release.Status = "push_failed"
			h.hub.Broadcast("status_changed", map[string]interface{}{"version": version, "status": "push_failed"})
		}
	}()

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Pushed to Git. Jenkins will deploy shortly.",
		"version": version,
	})
}

func (h *HelmReleaseHandler) DeleteHelmRelease(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	err := h.svc.DeleteHelmRelease(r.Context(), version)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	h.hub.Broadcast("release_deleted", map[string]interface{}{"version": version})
	w.WriteHeader(http.StatusNoContent)
}

func (h *HelmReleaseHandler) GetRecipes(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	recipes, err := h.svc.GetRecipesByHelmVersion(r.Context(), version)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, err.Error())
		return
	}
	h.respondWithJSON(w, http.StatusOK, recipes)
}

func (h *HelmReleaseHandler) AddRecipes(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	var recipe model.Recipe
	if err := json.NewDecoder(r.Body).Decode(&recipe); err != nil {
		h.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	added, err := h.svc.AddRecipeToRelease(r.Context(), version, recipe)
	if err != nil {
		h.respondWithError(w, http.StatusConflict, err.Error())
		return
	}

	h.hub.Broadcast("recipe_added", map[string]interface{}{"helmVersion": version, "recipe": added})
	h.respondWithJSON(w, http.StatusCreated, added)
}

func (h *HelmReleaseHandler) UpdateRecipe(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	recipeVersion := chi.URLParam(r, "recipeVersion")
	var recipe model.Recipe
	if err := json.NewDecoder(r.Body).Decode(&recipe); err != nil {
		h.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := h.svc.UpdateRecipeInRelease(r.Context(), version, recipeVersion, recipe)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	h.hub.Broadcast("recipe_updated", map[string]interface{}{"helmVersion": version, "recipe": updated})
	h.respondWithJSON(w, http.StatusOK, updated)
}

func (h *HelmReleaseHandler) DeleteRecipe(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	recipeVersion := chi.URLParam(r, "recipeVersion")
	err := h.svc.DeleteRecipeFromRelease(r.Context(), version, recipeVersion)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, err.Error())
		return
	}
	h.hub.Broadcast("recipe_deleted", map[string]interface{}{"helmVersion": version, "recipeVersion": recipeVersion})
	w.WriteHeader(http.StatusNoContent)
}

func (h *HelmReleaseHandler) GetComponents(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	recipeVersion := chi.URLParam(r, "recipeVersion")
	components, err := h.svc.GetComponentsByRecipe(r.Context(), version, recipeVersion)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, err.Error())
		return
	}
	h.respondWithJSON(w, http.StatusOK, components)
}

func (h *HelmReleaseHandler) GetUpgradePaths(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	recipeVersion := chi.URLParam(r, "recipeVersion")
	paths, err := h.svc.GetUpgradePaths(r.Context(), version, recipeVersion)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, err.Error())
		return
	}
	h.respondWithJSON(w, http.StatusOK, paths)
}

func (h *HelmReleaseHandler) CompareHelmVersions(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	result, err := h.svc.GetUpgradePathsBetweenHelmVersions(r.Context(), from, to)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.respondWithJSON(w, http.StatusOK, result)
}

func (h *HelmReleaseHandler) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	var upgrader = gorilla.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	h.hub.Register(conn)
	defer h.hub.Unregister(conn)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (h *HelmReleaseHandler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, map[string]string{
		"error": message,
	})
}

func (h *HelmReleaseHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
