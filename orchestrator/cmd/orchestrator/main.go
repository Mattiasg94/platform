package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"orchestrator/internal/config"
	"orchestrator/internal/gh"
	"orchestrator/internal/orchestrator"
	"orchestrator/internal/pod"
	"orchestrator/internal/repo"
	"orchestrator/internal/runs"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run boots the HTTP server and blocks until a shutdown signal, then drains. The
// orchestrator is now a long-lived Cloud Run service: it stays up and does a run
// when triggered, instead of running once and exiting. This is the shape the
// Temporal worker will need later, so nothing here is throwaway.
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	// Not "/healthz": Google's frontend intercepts that path and answers it itself,
	// so the request never reaches us. "/health" is ours.
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/run", handleRun)

	srv := &http.Server{
		Addr:              ":" + port(),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("orchestrator listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	// Cloud Run sends SIGTERM before it stops the instance; drain in-flight work
	// rather than cutting it mid-request. The window is deliberately short — a run
	// that outlives it is killed, which is the resilience we want, not a regression.
	log.Print("shutdown signal received; draining")
	drainCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(drainCtx)
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// handleRun triggers a run. Ingress auth is enforced by Cloud Run (the service
// requires an authenticated caller), so there is no token check here.
//
// The pipeline stays gated behind RUN_PIPELINE while the service's runtime env
// (project repo, GitHub token) is still being wired; until then a trigger proves
// the service is alive and returns a placeholder.
func handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	if os.Getenv("RUN_PIPELINE") != "1" {
		writeJSON(w, http.StatusAccepted, map[string]string{
			"status": "stub",
			"detail": "orchestrator reached — the run pipeline moves to the cloud in the next slice",
		})
		return
	}

	if err := dispatch(r.Context()); err != nil {
		log.Printf("run failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status": "error",
			"detail": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// dispatch drives one task to a merge-ready PR. It is the composition root, gated
// behind handleRun's flag until the service's runtime env is wired.
func dispatch(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	workspace, cleanup, err := repo.Checkout(ctx, cfg.ProjectRepoURL, cfg.ProjectRef, repo.Auth{Token: cfg.GitHubToken})
	if err != nil {
		return fmt.Errorf("checkout project repo: %w", err)
	}
	defer cleanup()

	blackboard, err := runs.NewStore(ctx, cfg.RunsBucket)
	if err != nil {
		return err
	}

	// The agent image is prebuilt per project by CI (agent-<project>); the runner
	// only names it — nothing is built here.
	runner, err := pod.NewCloudRun(ctx, workspace, cfg.ProjectRepo, cfg.RunsBucket, cfg.GCPProject, blackboard)
	if err != nil {
		return err
	}

	return orchestrator.Run(ctx, orchestrator.Deps{
		Runner:    runner,
		Forge:     gh.NewClient(cfg.GitHubToken, cfg.ProjectOwner, cfg.ProjectRepo),
		Workspace: workspace,
		BaseRef:   cfg.ProjectRef,
	})
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func port() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8080"
}
