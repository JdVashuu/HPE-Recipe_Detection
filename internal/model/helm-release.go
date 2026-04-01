package model

type HelmRelease struct {
	Version     string   `json:"version"`
	ReleaseName string   `json:"releaseName"`
	Status      string   `json:"status"`
	Recipes     []Recipe `json:"recipes"`
}
