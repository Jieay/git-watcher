package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	config "github.com/Jieay/git-watcher/configs"
	"github.com/Jieay/git-watcher/internal/git"
	"github.com/Jieay/git-watcher/internal/webhook"
)

// Scheduler manages periodic checks of Git repositories
type Scheduler struct {
	config        *config.ScheduleConfig
	gitManager    *git.Manager
	webhookClient *webhook.Client
	ticker        *time.Ticker
	mutex         sync.Mutex
	running       bool
	stopCh        chan struct{}
}

// NewScheduler creates a new scheduler
func NewScheduler(cfg *config.ScheduleConfig, gitManager *git.Manager, webhookClient *webhook.Client) *Scheduler {
	return &Scheduler{
		config:        cfg,
		gitManager:    gitManager,
		webhookClient: webhookClient,
		stopCh:        make(chan struct{}),
	}
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	if s.config.CheckInterval <= 0 {
		return fmt.Errorf("check interval must be greater than zero")
	}

	s.ticker = time.NewTicker(s.config.CheckInterval)
	s.running = true

	go func() {
		// Run once immediately
		s.runCheck()

		for {
			select {
			case <-s.ticker.C:
				s.runCheck()
			case <-s.stopCh:
				s.ticker.Stop()
				return
			case <-ctx.Done():
				s.ticker.Stop()
				return
			}
		}
	}()

	log.Printf("Scheduler started with interval: %v\n", s.config.CheckInterval)
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return
	}

	close(s.stopCh)
	s.running = false
	log.Println("Scheduler stopped")
}

// IsRunning returns true if the scheduler is running
func (s *Scheduler) IsRunning() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.running
}

// runCheck performs a check for repository updates on all configured branches
func (s *Scheduler) runCheck() {
	log.Println("Running scheduled check for repository updates on all configured branches")

	// Get configured branches
	gitConfig := s.gitManager.GetConfig()
	branches := gitConfig.Branches

	// Check each branch
	updatedBranches := make([]string, 0, len(branches))
	for _, branch := range branches {
		err := s.gitManager.CheckAndUpdateRepoBranch(branch)
		if err == nil {
			updatedBranches = append(updatedBranches, branch)
		} else {
			log.Printf("Error checking/updating branch %s: %v\n", branch, err)
		}
	}

	if len(updatedBranches) == 0 {
		log.Println("No branches were updated")
		return
	}

	// Create repository updates information
	repoUpdates := make(map[string]webhook.RepoUpdate)

	// Add an entry for each updated branch
	for _, branch := range updatedBranches {
		// Create a temporary copy of the main repo config with the correct branch
		repoCopy := &config.Repository{
			URL:       gitConfig.MainRepo.GetURL(),
			Branch:    branch,
			Directory: gitConfig.MainRepo.GetDirectory(),
			Auth:      gitConfig.MainRepo.GetAuth(),
		}

		// Get commit hash for the branch
		commitHash, err := s.gitManager.GetLastCommitHash(repoCopy)
		if err != nil {
			log.Printf("Warning: Could not get commit hash for branch %s: %v\n", branch, err)
			commitHash = "unknown"
		}

		repoUpdates[branch] = webhook.RepoUpdate{
			Repository: gitConfig.MainRepo.GetURL(),
			Branch:     branch,
			Timestamp:  time.Now(),
			CommitHash: commitHash,
		}
	}

	// Create webhook payload
	payload := webhook.WebhookPayload{
		Event:       "repository_update",
		Timestamp:   time.Now(),
		Message:     fmt.Sprintf("Updated %d branches: %v", len(updatedBranches), updatedBranches),
		RepoUpdates: repoUpdates,
	}

	// Send webhook notification
	if err := s.webhookClient.SendNotification(payload); err != nil {
		log.Printf("Error sending webhook notification: %v\n", err)
	}
}

// TriggerManualCheck triggers a manual check for repository updates on all branches
func (s *Scheduler) TriggerManualCheck() error {
	s.mutex.Lock()
	if !s.running {
		s.mutex.Unlock()
		return fmt.Errorf("scheduler is not running")
	}
	s.mutex.Unlock()

	go s.runCheck()
	return nil
}
