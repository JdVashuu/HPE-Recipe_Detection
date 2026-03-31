package test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/JdVashuu/RecipeDetection.git/internal/model"
)

func TestParsingJson(t *testing.T) {
	// 1. Read the actual file from the data folder
	// Note: Path is relative to the tests directory
	path := "../data/recipes_hpe.json"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read JSON file at %s: %v", path, err)
	}

	// 2. Unmarshal into the Catalog model
	var catalog model.Catalog
	err = json.Unmarshal(data, &catalog)
	if err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// 3. Verify the number of recipes found in the 'catalogs' array
	if len(catalog.Recipes) == 0 {
		t.Fatal("expected at least one recipe in catalogs, got 0")
	}

	// 4. Inspect the first recipe (Alletra MP Block - 10.48.703r.2501-GEN10)
	recipe := catalog.Recipes[0]
	if recipe.Version != "10.48.703r.2501-GEN10" {
		t.Errorf("expected version 10.48.703r.2501-GEN10, got %s", recipe.Version)
	}

	// 5. Verify the dynamic components map
	comps := recipe.UpdateComponents.Components

	// Check standard component (hypervisor)
	if hv, ok := comps["hypervisor"]; ok {
		expectedHV := "7.0.3-24411414"
		if hv.Version != expectedHV {
			t.Errorf("expected hypervisor version %s, got %s", expectedHV, hv.Version)
		}
	} else {
		t.Error("missing 'hypervisor' component in map")
	}

	// Check the flattened array (hypervisors -> hypervisors_0)
	if hv0, ok := comps["hypervisors_0"]; ok {
		expectedHV0 := "7.0.3-24411414"
		if hv0.Version != expectedHV0 {
			t.Errorf("expected hypervisors_0 version %s, got %s", expectedHV0, hv0.Version)
		}
	} else {
		t.Error("missing 'hypervisors_0' flattened component")
	}

	// Check another component (server_firmware)
	if sf, ok := comps["server_firmware"]; ok {
		if sf.Name != "Service Pack for ProLiant Gen10" {
			t.Errorf("wrong server_firmware name: %s", sf.Name)
		}
	} else {
		t.Error("missing 'server_firmware' component")
	}

	// 6. Demonstrate the success point (UpgradeTo)
	if len(recipe.UpgradeTo) > 0 {
		t.Logf("Success: UpgradeTo contains %d items, first is %s", len(recipe.UpgradeTo), recipe.UpgradeTo[0])
		if recipe.UpgradeTo[0] != "10.48.803g.2511-GEN10" {
			t.Errorf("expected first upgrade to be 10.48.803g.2511-GEN10, got %s", recipe.UpgradeTo[0])
		}
	} else {
		t.Error("UpgradeTo is empty - unmarshaling failed for this field")
	}
}
