package main

import "github.com/go-chi/chi/v5"

func (app *application) routes(r chi.Router) {
	// health
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", app.healthHandler.GetHealth)
	})

	//catalog endpoints
	r.Get("/catalogs", app.catalogHandler.GetAllCatalogs)
	r.Get("/catalogs/{catalogVersion}/recipes", app.catalogHandler.GetRecipeByCatalogs)

	// recipe endpoints
	r.Route("/recipes", func(r chi.Router) {
		r.Get("/{recipeVersion}/components", app.catalogHandler.GetComponentByRecipe)
		r.Get("/{recipeVersion}/upgradePaths", app.catalogHandler.GetUpgradePathsByRecipe)
	})

	// helm releases
	r.Route("/helm-release", func(r chi.Router) {
		r.Get("/", app.helmReleaseHandler.GetAllHelmReleases)
		r.Get("/{version}", app.helmReleaseHandler.GetHelmReleases)
		r.Post("/", app.helmReleaseHandler.CreateHelmRelease)
		r.Put("/{version}", app.helmReleaseHandler.UpdateHelmRelease)
		r.Put("/{version}/status", app.helmReleaseHandler.UpdateStatus)
		r.Post("/{version}/deploy", app.helmReleaseHandler.DeployRelease)
		r.Delete("/{version}", app.helmReleaseHandler.DeleteHelmRelease)

		// helm release recipe
		r.Get("/{version}/recipes", app.helmReleaseHandler.GetRecipes)
		r.Post("/{version}/recipes", app.helmReleaseHandler.AddRecipes)
		r.Put("/{version}/recipes/{recipeVersion}", app.helmReleaseHandler.UpdateRecipe)
		r.Delete("/{version}", app.helmReleaseHandler.DeleteRecipe)

		// helm release recipe components and upgrades
		r.Get("/{version}/recipes/{recipeVersion}/components", app.helmReleaseHandler.GetComponents)
		r.Get("/{version}/recipes/{recipeVersion}/upgradePaths", app.helmReleaseHandler.GetUpgradePaths)
	})

	// helm comparsion and Websockets
	r.Get("/helm-releases/compare", app.helmReleaseHandler.CompareHelmVersions)
	r.Get("/ws", app.helmReleaseHandler.WebSocketHandler)
}
