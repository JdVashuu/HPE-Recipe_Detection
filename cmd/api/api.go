package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/JdVashuu/RecipeDetection.git/internal/env"
	gitcfg "github.com/JdVashuu/RecipeDetection.git/internal/git"
	"github.com/JdVashuu/RecipeDetection.git/internal/handler"
	"github.com/JdVashuu/RecipeDetection.git/internal/repository"
	"github.com/JdVashuu/RecipeDetection.git/internal/service"
	"github.com/JdVashuu/RecipeDetection.git/internal/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type config struct {
	addr   string
	GitOps gitcfg.GitOps
}

type application struct {
	config config

	catalogSvc     *service.CatalogService
	helmReleaseSvc *service.HelmReleaseService
	gitopsSvc      *service.GitOpsService

	catalogHandler     *handler.CatalogHandler
	healthHandler      *handler.HealthHandler
	helmReleaseHandler *handler.HelmReleaseHandler

	hub       *websocket.Hub
	k8sClient *kubernetes.Clientset
}

func (app *application) setupK8sClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := env.GetString("KUBECONFIG", (os.Getenv("HOME") + "/.kube/config"))
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}
	return kubernetes.NewForConfig(config)
}

func corsMiddeware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (app *application) initialise() error {
	k8sClient, err := app.setupK8sClient()
	if err != nil {
		return err
	}
	app.k8sClient = k8sClient

	k8sRepo := repository.NewK8sCMRepo(k8sClient, "default")

	app.catalogSvc = service.NewCatalogService()
	app.helmReleaseSvc = service.NewHelmReleaseService(k8sRepo)
	app.gitopsSvc = service.NewGitOpsService(&app.config.GitOps)

	app.hub = websocket.NewHub()
	go app.hub.Run()

	app.healthHandler = handler.NewHealthHandler()
	app.catalogHandler = handler.NewCatalogHandler(app.catalogSvc)
	app.helmReleaseHandler = handler.NewHelmReleaseHandler(app.helmReleaseSvc, app.gitopsSvc, app.hub)

	return nil
}

func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(corsMiddeware)

	app.routes(r)

	return r
}

func (app *application) run(mux http.Handler) error {
	srv := &http.Server{
		Addr:    app.config.addr,
		Handler: mux,
	}

	log.Printf("The server has started at %s", app.config.addr)
	return srv.ListenAndServe()
}
