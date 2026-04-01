package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/JdVashuu/RecipeDetection.git/internal/model"
	"github.com/JdVashuu/RecipeDetection.git/internal/repository"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	LabelAppName          = "app.kubernetes.io/name"
	LabelAppVersion       = "app.kubernetes.io/version"
	AnnotationReleaseName = "meta.helm.sh/release-name"
	RecipeDataKey         = "recipe-data.json"
)

type HelmReleaseService struct {
	repo *repository.K8sConfigMapRepository
}

func NewHelmReleaseService(repo *repository.K8sConfigMapRepository) *HelmReleaseService {
	return &HelmReleaseService{
		repo: repo,
	}
}

func (s *HelmReleaseService) parseConfigMap(cm corev1.ConfigMap) (*model.HelmRelease, error) {
	jsonData, ok := cm.Data[RecipeDataKey]
	if !ok || jsonData == "" {
		return nil, fmt.Errorf("no recipe data found in ConfigMap %s", cm.Name)
	}

	var data struct {
		ChartVersion string         `json:"chart_version"`
		Recipes      []model.Recipe `json:"recipes"`
	}

	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		return nil, err
	}

	releaseName := cm.Annotations[AnnotationReleaseName]
	if releaseName == "" {
		releaseName = "unknown"
	}

	return &model.HelmRelease{
		Version:     data.ChartVersion,
		ReleaseName: releaseName,
		Status:      "deployed",
		Recipes:     data.Recipes,
	}, nil
}

func (s *HelmReleaseService) GetAllHelmRelease(ctx context.Context) ([]model.HelmRelease, error) {
	cms, err := s.repo.ListRecipeConfigMaps(ctx)
	if err != nil {
		return nil, err
	}

	byVersion := make(map[string]model.HelmRelease)
	for _, cm := range cms {
		release, err := s.parseConfigMap(cm)
		if err != nil && release != nil {
			if _, exists := byVersion[release.Version]; !exists {
				byVersion[release.Version] = *release
			}
		}
	}

	var releases []model.HelmRelease
	for _, r := range byVersion {
		releases = append(releases, r)
	}

	sort.Slice(releases, func(i, j int) bool {
		return releases[i].Version < releases[j].Version
	})
	return releases, nil
}

func (s *HelmReleaseService) GetHelmRelease(ctx context.Context, version string) (*model.HelmRelease, error) {
	releases, err := s.GetAllHelmRelease(ctx)
	if err != nil {
		return nil, err
	}

	for _, release := range releases {
		if release.Version == version {
			return &release, nil
		}
	}

	return nil, fmt.Errorf("Helm release %s not found", version)
}

func (s *HelmReleaseService) CreateHelmRelease(ctx context.Context, release *model.HelmRelease) (*model.HelmRelease, error) {
	existing, _ := s.GetHelmRelease(ctx, release.Version)
	if existing != nil {
		return nil, fmt.Errorf("Helm release %s already exists", release.Version)
	}

	if release.Recipes == nil {
		release.Recipes = []model.Recipe{}
	}

	if release.Status == "" {
		release.Status = "pending"
	}

	jsonData, err := s.buildRecipeJSON(release)
	if err != nil {
		return nil, err
	}

	cmName := fmt.Sprintf("recipe-v%s-config", strings.ReplaceAll(release.Version, ".", "-"))
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: "default",
			Labels: map[string]string{
				LabelAppName:                   "recipe-detection",
				LabelAppVersion:                release.Version,
				"app.kubernetes.io/managed-by": "recipe-detection-api",
			},
			Annotations: map[string]string{
				AnnotationReleaseName: release.ReleaseName,
			},
		},
		Data: map[string]string{
			"chart-version": release.Version,
			RecipeDataKey:   jsonData,
		},
	}

	if release.ReleaseName == "" {
		cm.Annotations[AnnotationReleaseName] = fmt.Sprintf("recipe-v%s", strings.ReplaceAll(release.Version, ".", "-"))
	}

	_, err = s.repo.CreateConfigMap(ctx, cm)
	if err != nil {
		return nil, err
	}

	return release, nil
}

func (s *HelmReleaseService) UpdateHelmRelease(ctx context.Context, version string, updated *model.HelmRelease) (*model.HelmRelease, error) {
	existing, err := s.GetHelmRelease(ctx, version)
	if err != nil {
		return nil, err
	}

	if updated.ReleaseName != "" {
		existing.ReleaseName = updated.ReleaseName
	}

	if updated.Status != "" {
		existing.Status = updated.Status
	}

	if updated.ReleaseName != "" {
		existing.Status = updated.Status
	}

	err = s.updateConfigMap(ctx, version, existing)
	if err != nil {
		return nil, err
	}

	return existing, nil
}

func (s *HelmReleaseService) DeleteHelmRelease(ctx context.Context, version string) error {
	cms, err := s.repo.ListRecipeConfigMaps(ctx)
	if err != nil {
		return err
	}

	for _, cm := range cms {
		release, err := s.parseConfigMap(cm)
		if err == nil && release != nil && release.Version == version {
			return s.repo.DeleteConfigMap(ctx, cm.Name)
		}
	}

	return fmt.Errorf("helm release %s not found", version)
}

func (s *HelmReleaseService) AddRecipeToRelease(ctx context.Context, helmVersion string, recipe model.Recipe) (*model.Recipe, error) {
	release, err := s.GetHelmRelease(ctx, helmVersion)
	if err != nil {
		return nil, err
	}

	for _, r := range release.Recipes {
		if r.Version == recipe.Version {
			return nil, fmt.Errorf("recipe %s already exists in helm release %s", release.Version, helmVersion)
		}
	}

	if len(recipe.UpdateComponents.Components) == 0 {
		recipe.UpdateComponents = model.UpdateComponents{}
	}

	if recipe.UpgradeTo == nil {
		recipe.UpgradeTo = []string{}
	}

	release.Recipes = append(release.Recipes, recipe)
	err = s.updateConfigMap(ctx, helmVersion, release)
	if err != nil {
		return nil, err
	}

	return &recipe, nil
}

func (s *HelmReleaseService) UpdateRecipeInRelease(ctx context.Context, helmVersion, recipeVersion string, updated model.Recipe) (*model.Recipe, error) {
	release, err := s.GetHelmRelease(ctx, helmVersion)
	if err != nil {
		return nil, err
	}

	var found *model.Recipe
	for i, r := range release.Recipes {
		if r.Version == recipeVersion {
			if updated.ReleaseDate != "" {
				release.Recipes[i].ReleaseDate = updated.ReleaseDate
			}
			if updated.Retired != "" {
				release.Recipes[i].Retired = updated.Retired
			}
			if updated.ServerModelGenType != "" {
				release.Recipes[i].ServerModelGenType = updated.ServerModelGenType
			}
			if updated.State != "" {
				release.Recipes[i].State = updated.State
			}
			if updated.StorageModel != "" {
				release.Recipes[i].StorageModel = updated.StorageModel
			}
			if updated.UpgradeTo == nil {
				release.Recipes[i].UpgradeTo = updated.UpgradeTo
			}
			if updated.UpdateComponents.Components == nil {
				release.Recipes[i].UpdateComponents = updated.UpdateComponents
			}
			found = &release.Recipes[i]
			break
		}
	}

	if found == nil {
		return nil, fmt.Errorf("recipe %s not found in helm release %s", recipeVersion, helmVersion)
	}

	err = s.updateConfigMap(ctx, helmVersion, release)
	if err != nil {
		return nil, err
	}

	return found, nil
}

func (s *HelmReleaseService) DeleteRecipeFromRelease(ctx context.Context, helmVersion, recipeVersion string) error {
	release, err := s.GetHelmRelease(ctx, helmVersion)
	if err != nil {
		return err
	}

	newRecipes := []model.Recipe{}
	found := false
	for _, r := range release.Recipes {
		if r.Version == recipeVersion {
			found = true
			continue
		}
		newRecipes = append(newRecipes, r)
	}

	if !found {
		return fmt.Errorf("recipe %s not found in helm release %s", recipeVersion, helmVersion)
	}

	release.Recipes = newRecipes
	return s.updateConfigMap(ctx, helmVersion, release)
}

func (s *HelmReleaseService) GetRecipesByHelmVersion(ctx context.Context, version string) ([]model.Recipe, error) {
	release, err := s.GetHelmRelease(ctx, version)
	if err != nil {
		return nil, err
	}
	return release.Recipes, nil
}

func (s *HelmReleaseService) GetComponentsByRecipe(ctx context.Context, helmVersion, recipeVersion string) (model.UpdateComponents, error) {
	recipes, err := s.GetRecipesByHelmVersion(ctx, helmVersion)
	if err != nil {
		return model.UpdateComponents{}, err
	}

	for _, r := range recipes {
		if r.Version == recipeVersion {
			return r.UpdateComponents, nil
		}
	}

	return model.UpdateComponents{}, fmt.Errorf("recipe %s not found in helm release %s", recipeVersion, helmVersion)
}

func (s *HelmReleaseService) GetUpgradePaths(ctx context.Context, helmVersion, recipeVersion string) ([]string, error) {
	recipes, err := s.GetRecipesByHelmVersion(ctx, helmVersion)
	if err != nil {
		return nil, err
	}

	for _, r := range recipes {
		if r.Version == recipeVersion {
			return r.UpgradeTo, nil
		}
	}

	return nil, fmt.Errorf("recipe %s not found in helm release %s", recipeVersion, helmVersion)
}

func (s *HelmReleaseService) GetUpgradePathsBetweenHelmVersions(ctx context.Context, fromVersion, toVersion string) (map[string]interface{}, error) {
	fromRelease, err := s.GetHelmRelease(ctx, fromVersion)
	if err != nil {
		return nil, err
	}
	toRelease, err := s.GetHelmRelease(ctx, toVersion)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"fromHelmVersion": fromVersion,
		"toHelmVersion":   toVersion,
	}

	fromRecipeVersions := []string{}
	for _, r := range fromRelease.Recipes {
		fromRecipeVersions = append(fromRecipeVersions, r.Version)
	}

	toRecipeVersions := []string{}
	for _, r := range toRelease.Recipes {
		toRecipeVersions = append(toRecipeVersions, r.Version)
	}

	removedRecipes := []string{}
	for _, v := range fromRecipeVersions {
		found := false
		for _, tv := range toRecipeVersions {
			if v == tv {
				found = true
				break
			}
		}
		if !found {
			removedRecipes = append(removedRecipes, v)
		}
	}

	addedRecipes := []string{}
	for _, v := range toRecipeVersions {
		found := false
		for _, fv := range fromRecipeVersions {
			if v == fv {
				found = true
				break
			}
		}
		if !found {
			addedRecipes = append(addedRecipes, v)
		}
	}

	recipeChanges := map[string]interface{}{
		"removed": removedRecipes,
		"added":   addedRecipes,
	}
	result["recipeChanges"] = recipeChanges

	latestFrom := fromRelease.Recipes[len(fromRelease.Recipes)-1]
	latestTo := toRelease.Recipes[len(toRelease.Recipes)-1]

	componentDiffs := make(map[string]interface{})
	allComponents := make(map[string]bool)
	for k := range latestFrom.UpdateComponents.Components {
		allComponents[k] = true
	}
	for k := range latestTo.UpdateComponents.Components {
		allComponents[k] = true
	}

	for component := range allComponents {
		fromVer, okFrom := latestFrom.UpdateComponents.Components[component]
		fromVerStr := ""
		if okFrom {
			fromVerStr = fromVer.ReleaseID
		} else {
			fromVerStr = "N/A"
		}

		toVer, okTo := latestTo.UpdateComponents.Components[component]
		toVerStr := ""
		if okTo {
			toVerStr = toVer.ReleaseID
		} else {
			toVerStr = "N/A"
		}

		if fromVerStr != toVerStr {
			componentDiffs[component] = map[string]string{
				"from": fromVerStr,
				"to":   toVerStr,
			}
		}
	}
	result["componentVersionDiffs"] = componentDiffs

	return result, nil
}

func (s *HelmReleaseService) updateConfigMap(ctx context.Context, chartVersion string, release *model.HelmRelease) error {
	cms, err := s.repo.ListRecipeConfigMaps(ctx)
	if err != nil {
		return err
	}

	for _, cm := range cms {
		parsed, err := s.parseConfigMap(cm)
		if err == nil && parsed != nil && parsed.Version == chartVersion {
			jsonData, err := s.buildRecipeJSON(release)
			if err != nil {
				return err
			}
			cm.Data[RecipeDataKey] = jsonData
			_, err = s.repo.UpdateConfigMap(ctx, &cm)
			return err
		}
	}

	return fmt.Errorf("ConfigMap for helm release %s not found", chartVersion)
}

func (s *HelmReleaseService) buildRecipeJSON(release *model.HelmRelease) (string, error) {
	data := map[string]interface{}{
		"chartVersion": release.Version,
		"recipes":      release.Recipes,
	}
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
