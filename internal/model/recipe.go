package model

type Recipe struct {
	Version            string           `json:"catalog_version"`
	ReleaseDate        string           `json:"release_date"`
	Retired            string           `json:"retired"`
	ServerModelGenType string           `json:"server_model_gen_type"`
	State              string           `json:"state"`
	StorageModel       string           `json:"storage_model"`
	UpdateComponents   UpdateComponents `json:"update_components"`
	UpgradeTo          []string         `json:"upgrade_to"`
}
