package service

import "github.com/JdVashuu/RecipeDetection.git/internal/repository"

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
