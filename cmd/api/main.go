package main

import (
	"log"

	"github.com/JdVashuu/RecipeDetection.git/internal/env"
	gitcfg "github.com/JdVashuu/RecipeDetection.git/internal/git"
)

func main() {
	cfg := config{
		addr:   env.GetString("ADDR", ":8080"),
		GitOps: *gitcfg.LoadGitOpsConfig(),
	}

	app := &application{
		config: cfg,
	}

	mux := app.mount()
	log.Fatal(app.run(mux))
}
