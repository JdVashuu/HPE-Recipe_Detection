package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	gitcfg "github.com/JdVashuu/RecipeDetection.git/internal/git"
	"github.com/JdVashuu/RecipeDetection.git/internal/model"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"go.yaml.in/yaml/v2"
)

type GitOpsService struct {
	cfg *gitcfg.GitOps
	mux sync.Mutex
}

func NewGitOpsService(cfg *gitcfg.GitOps) *GitOpsService {
	return &GitOpsService{
		cfg: cfg,
	}
}

func (s *GitOpsService) GenerateAndPush(release *model.HelmRelease) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	repo, err := s.getOrCloneRepo()
	if err != nil {
		return err
	}

	wt, err := repo.Worktree()
	if err != nil {
		return err
	}

	//git pull
	err = wt.Pull(&git.PullOptions{
		RemoteName:    "origin",
		ReferenceName: plumbing.NewBranchReferenceName(s.cfg.Branch),
		Auth:          s.getAuth(),
		SingleBranch:  true,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}

	// gen values
	valuesYaml, err := s.generateValuesYaml(release)
	if err != nil {
		return err
	}

	valuesFileName := fmt.Sprintf("values-v%s.yaml", release.Version)
	valuesFilePath := filepath.Join(s.cfg.ValuesDir, valuesFileName)
	fullValuesPath := filepath.Join(s.cfg.LocalPath, valuesFilePath)

	err = os.Mkdir(filepath.Dir(fullValuesPath), 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile(fullValuesPath, []byte(valuesYaml), 0644)
	if err != nil {
		return err
	}

	// update chart
	chartFilePath := filepath.Join(s.cfg.ValuesDir, "Chart.yaml")
	fullChartPath := filepath.Join(s.cfg.LocalPath, chartFilePath)
	err = s.updateChartVersion(fullChartPath, release.Version)
	if err != nil {
		return err
	}

	// stage
	_, err = wt.Add(valuesFilePath)
	if err != nil {
		return err
	}

	_, err = wt.Add(chartFilePath)
	if err != nil {
		return err
	}

	// Commit
	commitMsg := fmt.Sprintf("Release v%s: Update recipe values\n\nRecipes; %d recipe(s)\nTriggered from Recipe Detection UI",
		release.Version, len(release.Recipes))
	_, err = wt.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Recipe Detection",
			Email: "recipe-detection@hpe.com",
		},
	})
	if err != nil {
		return err
	}

	// push
	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       s.getAuth(),
	})
	if err != nil {
		return nil
	}

	return nil
}

func (s *GitOpsService) getOrCloneRepo() (*git.Repository, error) {
	repo, err := git.PlainOpen(s.cfg.LocalPath)
	if err != nil {
		return repo, err
	}

	return git.PlainClone(s.cfg.LocalPath, false, &git.CloneOptions{
		URL:           s.cfg.RepoURL,
		ReferenceName: plumbing.NewBranchReferenceName(s.cfg.Branch),
		Auth:          s.getAuth(),
		SingleBranch:  true,
	})
}

func (s *GitOpsService) getAuth() *http.BasicAuth {
	if s.cfg.Token == "" {
		return nil
	}

	return &http.BasicAuth{
		Username: s.cfg.Username,
		Password: s.cfg.Token,
	}
}

func (s *GitOpsService) generateValuesYaml(release *model.HelmRelease) (string, error) {
	type RecipeData struct {
		chartVersion string       `yaml:"chartVersion"`
		Recipes      model.Recipe `yaml:"recipe"`
	}
	type Root struct {
		RecipeData RecipeData `yaml:"recipeData"`
	}

	root := Root{
		RecipeData: RecipeData{
			chartVersion: release.Version,
			Recipes:      release.Recipes,
		},
	}

	data, err := yaml.Marshal(&root)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (s *GitOpsService) updateChartVersion(chartPath, version string) error {
	content, err := os.ReadFile(chartPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "version:") {
			lines[i] = fmt.Sprintf("version: %s", version)
		} else if strings.HasPrefix(line, "appVersion:") {
			lines[i] = fmt.Sprintf("appVersion: \"%s\"", version)
		}
	}
	return os.WriteFile(chartPath, []byte(strings.Join(lines, "\n")), 0644)

}
