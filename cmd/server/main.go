package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	config "github.com/Jieay/git-watcher/configs"
	"github.com/Jieay/git-watcher/internal/git"
	"github.com/Jieay/git-watcher/internal/scheduler"
	"github.com/Jieay/git-watcher/internal/webhook"
)

var (
	configFile = flag.String("config", "configs/config.local.json", "Path to configuration file")
)

// handleArtifactsWebhook handles the artifacts webhook
func handleArtifactsWebhook(gitManager *git.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read and parse the request body
		var payload struct {
			Artifact struct {
				UserId              int64   `json:"userId"`
				UserName            string  `json:"userName"`
				ProjectId           int64   `json:"projectId"`
				ProjectName         string  `json:"projectName"`
				TeamId              int64   `json:"teamId"`
				Action              string  `json:"action"`
				ArtifactType        string  `json:"artifactType"`
				ArtifactRepoId      int64   `json:"artifactRepoId"`
				ArtifactRepoName    string  `json:"artifactRepoName"`
				ArtifactPkgId       int64   `json:"artifactPkgId"`
				ArtifactPkgName     string  `json:"artifactPkgName"`
				ArtifactVersionId   int64   `json:"artifactVersionId"`
				ArtifactVersionName string  `json:"artifactVersionName"`
				Size                float64 `json:"size"`
			} `json:"artifact"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if payload.Artifact.ArtifactRepoName == "" || payload.Artifact.ArtifactPkgName == "" || payload.Artifact.ArtifactVersionName == "" {
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		// Update the artifacts repository
		if err := gitManager.UpdateArtifactsRepo(
			payload.Artifact.ArtifactRepoName,
			payload.Artifact.ArtifactPkgName,
			payload.Artifact.ArtifactVersionName,
		); err != nil {
			http.Error(w, fmt.Sprintf("Failed to update artifacts: %v", err), http.StatusInternalServerError)
			return
		}

		// Prepare response with more detailed information
		response := map[string]interface{}{
			"status":  "success",
			"message": fmt.Sprintf("Successfully updated artifacts for %s", payload.Artifact.ArtifactRepoName),
			"details": map[string]interface{}{
				"userId":              payload.Artifact.UserId,
				"userName":            payload.Artifact.UserName,
				"projectId":           payload.Artifact.ProjectId,
				"projectName":         payload.Artifact.ProjectName,
				"teamId":              payload.Artifact.TeamId,
				"action":              payload.Artifact.Action,
				"artifactType":        payload.Artifact.ArtifactType,
				"artifactRepoId":      payload.Artifact.ArtifactRepoId,
				"artifactRepoName":    payload.Artifact.ArtifactRepoName,
				"artifactPkgId":       payload.Artifact.ArtifactPkgId,
				"artifactPkgName":     payload.Artifact.ArtifactPkgName,
				"artifactVersionId":   payload.Artifact.ArtifactVersionId,
				"artifactVersionName": payload.Artifact.ArtifactVersionName,
				"size":                payload.Artifact.Size,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Printf("Warning: Failed to load configuration file: %v", err)
		// 如果配置文件加载失败，使用默认配置
		cfg = &config.Config{
			Server: config.ServerConfig{
				Port: 8080, // 默认端口
			},
			Git: config.GitConfig{
				MainRepo: &config.Repository{
					Branch: "main",
				},
				ArtifactsRepo: &config.ArtifactsRepo{
					Branch: "main",
				},
			},
			Schedule: config.ScheduleConfig{
				CheckInterval: 10 * time.Minute,
			},
		}
	}

	// 确保环境变量覆盖配置文件
	config.OverrideWithEnv(cfg)

	// Initialize Git manager
	gitManager, err := git.NewManager(&cfg.Git)
	if err != nil {
		log.Fatalf("Failed to initialize Git manager: %v", err)
	}

	// Initialize webhook client
	webhookClient := webhook.NewClient(&cfg.Webhook)

	// Initialize scheduler
	sched := scheduler.NewScheduler(&cfg.Schedule, gitManager, webhookClient)

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the scheduler
	if err := sched.Start(ctx); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}

	// Set up HTTP server
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// Webhook endpoint to trigger manual check
	mux.HandleFunc("/webhook/trigger", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		payload, err := webhookClient.ValidateWebhook(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid webhook request: %v", err), http.StatusBadRequest)
			return
		}

		// 打印接收到的所有字段信息
		log.Printf("=== Webhook Trigger Received ===")
		log.Printf("Event: %s", payload.Event)
		log.Printf("Branch: %s", payload.Branch)
		log.Printf("Reference: %s", payload.Reference)
		log.Printf("Ref: %s", payload.Ref)
		log.Printf("===============================")

		logMsg := "Received webhook trigger -"
		logMsg += fmt.Sprintf(" Event: %s,", payload.Event)

		if payload.Branch != "" {
			logMsg += fmt.Sprintf(" Branch: %s,", payload.Branch)
		}
		if payload.Reference != "" {
			logMsg += fmt.Sprintf(" Reference: %s,", payload.Reference)
		}
		if payload.Ref != "" {
			logMsg += fmt.Sprintf(" Ref: %s,", payload.Ref)
		}
		// Trim trailing comma if exists
		logMsg = strings.TrimSuffix(logMsg, ",")
		log.Print(logMsg)

		// If a specific branch is provided, check only that branch
		if payload.Branch != "" {
			if err := gitManager.CheckAndUpdateRepoBranch(payload.Branch); err != nil {
				http.Error(w, fmt.Sprintf("Failed to update branch %s: %v", payload.Branch, err), http.StatusInternalServerError)
				return
			}

			// Create webhook payload for notification
			mainRepoHash, _ := gitManager.GetLastCommitHash(gitManager.GetConfig().MainRepo)
			repoUpdates := make(map[string]webhook.RepoUpdate)
			repoUpdates["main"] = webhook.RepoUpdate{
				Repository: gitManager.GetConfig().MainRepo.GetURL(),
				Branch:     payload.Branch,
				Timestamp:  time.Now(),
				CommitHash: mainRepoHash,
			}

			notifyPayload := webhook.WebhookPayload{
				Event:       "repository_update",
				Timestamp:   time.Now(),
				Branch:      payload.Branch,
				Message:     fmt.Sprintf("Repository branch %s and submodules update completed", payload.Branch),
				RepoUpdates: repoUpdates,
			}

			// Send webhook notification
			if err := webhookClient.SendNotification(notifyPayload); err != nil {
				log.Printf("Error sending webhook notification: %v", err)
			}

			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Manual check for branch %s completed", payload.Branch)
			return
		}

		// If no branch specified, trigger check for all branches
		if err := sched.TriggerManualCheck(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to trigger check: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Manual check for all branches triggered")
	})

	// Status endpoint
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		isRunning := sched.IsRunning()
		status := "running"
		if !isRunning {
			status = "stopped"
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Scheduler status: %s", status)
	})

	// Add the new artifacts webhook route
	mux.HandleFunc("/webhook/artifacts", handleArtifactsWebhook(gitManager))

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("Starting server on port %d", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	<-sigChan
	log.Println("Shutting down server...")

	// Create a deadline to wait for
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Stop the scheduler
	sched.Stop()

	// Shutdown the server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server gracefully stopped")
}
