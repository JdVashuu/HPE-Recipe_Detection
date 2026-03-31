package model

import (
	"encoding/json"
	"fmt"
)

type Component struct {
	BinaryDownloadURL string   `json:"binary_download_url"`
	EulaURL           string   `json:"eula_url"`
	FileType          string   `json:"file_type"`
	Installer         string   `json:"installer"`
	IsBlacklisted     bool     `json:"is_blacklisted"`
	Name              string   `json:"name"`
	ProfileName       string   `json:"profile_name,omitempty"`
	ReleaseDate       string   `json:"release_date"`
	ReleaseID         string   `json:"release_id"`
	ReleaseNotesURL   string   `json:"release_notes_url"`
	ReleaseSize       string   `json:"release_size"`
	ReleaseStatus     string   `json:"release_status"`
	ServerModel       string   `json:"server_model,omitempty"`
	Signature         string   `json:"signature"`
	Type              string   `json:"type,omitempty"`
	UpgradeFrom       []string `json:"upgrade_from"`
	Version           string   `json:"version"`
	ValidPath         bool     `json:"valid_path,omitempty"`
}

type UpdateComponents struct {
	Components map[string]Component `json:"-"`
}

func (u *UpdateComponents) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	u.Components = make(map[string]Component)

	for key, value := range raw {

		// special case: array
		if key == "hypervisors" {
			var list []Component
			if err := json.Unmarshal(value, &list); err != nil {
				return err
			}

			for i, c := range list {
				u.Components[fmt.Sprintf("%s_%d", key, i)] = c
			}
			continue
		}

		var comp Component
		if err := json.Unmarshal(value, &comp); err != nil {
			return err
		}

		u.Components[key] = comp
	}

	return nil
}
