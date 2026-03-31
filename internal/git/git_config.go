package gitcfg

import (
	"github.com/JdVashuu/RecipeDetection.git/internal/env"
)

type GitOps struct {
	RepoURL   string
	LocalPath string
	Branch    string
	Username  string
	Token     string
	ValuesDir string
}

func LoadGitOpsConfig() *GitOps {
	gcfg := &GitOps{
		RepoURL:   env.GetString("GITOPS_REPO_URL", "https://github.com/JdVashuu/helm-chart-recipies.git"),
		LocalPath: env.GetString("GITOPS_LOCAL_PATH", "/tmp/recipe-detection-helm"),
		Branch:    env.GetString("GITOPS_BRANCH", "main"),
		Username:  env.GetString("GITOPS_USERNAME", "JdVashuu"),
		Token:     env.GetString("GITOPS_TOKEN", ""),
		ValuesDir: env.GetString("GITOPS_VALUES_DIR", "helm/sample-chart"),
	}

	return gcfg
}
