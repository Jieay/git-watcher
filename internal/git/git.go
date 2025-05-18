package git

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	config "github.com/Jieay/git-watcher/configs"
)

// Manager handles Git operations
type Manager struct {
	config *config.GitConfig
	// 添加文件锁和 Git 操作锁
	fileLocks    map[string]*sync.Mutex
	gitOpLock    sync.Mutex
	fileLocksMux sync.Mutex
}

// NewManager creates a new Git manager
func NewManager(cfg *config.GitConfig) (*Manager, error) {
	// Ensure the working directory exists
	if err := os.MkdirAll(cfg.WorkingDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create working directory: %w", err)
	}

	return &Manager{
		config:    cfg,
		fileLocks: make(map[string]*sync.Mutex),
	}, nil
}

// getFileLock 获取指定文件的锁
func (m *Manager) getFileLock(filePath string) *sync.Mutex {
	m.fileLocksMux.Lock()
	defer m.fileLocksMux.Unlock()

	lock, exists := m.fileLocks[filePath]
	if !exists {
		lock = &sync.Mutex{}
		m.fileLocks[filePath] = lock
	}
	return lock
}

// CheckAndUpdateRepos checks for updates in the main repository and its submodules
// for all configured branches
func (m *Manager) CheckAndUpdateRepos() error {
	for _, branch := range m.config.Branches {
		if err := m.CheckAndUpdateRepoBranch(branch); err != nil {
			return fmt.Errorf("failed to check/update branch %s: %w", branch, err)
		}
	}
	return nil
}

// CheckAndUpdateRepoBranch checks for updates in the main repository for a specific branch
func (m *Manager) CheckAndUpdateRepoBranch(branch string) error {
	// Create a copy of the main repo config with the specified branch
	repoCopy := &config.Repository{
		URL:       m.config.MainRepo.GetURL(),
		Branch:    branch,
		Directory: m.config.MainRepo.GetDirectory(),
		Auth:      m.config.MainRepo.GetAuth(),
	}

	// Check and update the main repository for the specified branch
	mainRepoUpdated, err := m.checkAndUpdateRepo(repoCopy)
	if err != nil {
		return fmt.Errorf("failed to check/update main repo %s branch %s: %w",
			m.config.MainRepo.GetURL(), branch, err)
	}

	if !m.config.UseSubmodules {
		return nil
	}

	// If using submodules and main repo updated, update all submodules
	var submodulesUpdated bool
	if mainRepoUpdated {
		if err := m.updateSubmodules(); err != nil {
			return fmt.Errorf("failed to update submodules: %w", err)
		}
		fmt.Printf("Successfully updated main repository branch %s and all submodules\n", branch)
		submodulesUpdated = true
	} else {
		// Even if main repo wasn't updated, check submodules for updates
		submodulesUpdated, err = m.checkAndUpdateSubmodules()
		if err != nil {
			return fmt.Errorf("failed to check and update submodules: %w", err)
		}
	}

	// If auto commit is enabled and there were updates to submodules,
	// commit those changes to the main repository
	if m.config.AutoCommit && submodulesUpdated {
		if err := m.commitSubmoduleChangesToMainRepo(branch); err != nil {
			return fmt.Errorf("failed to commit submodule changes to main repository: %w", err)
		}
	}

	return nil
}

// checkAndUpdateSubmodules checks if any submodules have updates and updates them if they do
// Returns true if any submodules were updated
func (m *Manager) checkAndUpdateSubmodules() (bool, error) {
	repoPath := filepath.Join(m.config.WorkingDir, m.config.MainRepo.GetDirectory())
	gitmodulesPath := filepath.Join(repoPath, ".gitmodules")

	// Check if .gitmodules exists
	if _, err := os.Stat(gitmodulesPath); os.IsNotExist(err) {
		fmt.Println("No .gitmodules file found")
		return false, nil
	}

	// Get list of submodules
	submodules, err := m.listSubmodules()
	if err != nil {
		return false, fmt.Errorf("failed to list submodules: %w", err)
	}

	if len(submodules) == 0 {
		fmt.Println("No submodules found in .gitmodules")
		return false, nil
	}

	fmt.Printf("Found %d submodules: %v\n", len(submodules), submodules)

	// Check status of all submodules
	statusCmd := exec.Command("git", "submodule", "status")
	statusCmd.Dir = repoPath
	statusOutput, err := statusCmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to check submodule status: %w", err)
	}

	fmt.Printf("Submodule status:\n%s\n", string(statusOutput))

	var anyUpdated bool
	for _, submodule := range submodules {
		// Check if submodule needs updating
		updateCmd := exec.Command("git", "submodule", "update", "--recursive", "--remote", submodule)
		updateCmd.Dir = repoPath
		if m.config.MainRepo.GetAuth().Type == "ssh" && m.config.MainRepo.GetAuth().SSHKeyPath != "" {
			updateCmd.Env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no", m.config.MainRepo.GetAuth().SSHKeyPath))
		}
		output, err := updateCmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Warning: Failed to update submodule %s: %v\nOutput: %s\n", submodule, err, string(output))
			continue
		}

		// Get the new commit hash after update
		hashCmd := exec.Command("git", "rev-parse", "HEAD")
		hashCmd.Dir = filepath.Join(repoPath, submodule)
		hashOutput, err := hashCmd.Output()
		if err != nil {
			fmt.Printf("Warning: Failed to get commit hash for submodule %s: %v\n", submodule, err)
			continue
		}

		commitHash := strings.TrimSpace(string(hashOutput))
		fmt.Printf("Updated submodule %s to commit %s\n", submodule, commitHash)
		anyUpdated = true
	}

	// Check status of main repository after updates
	mainStatusCmd := exec.Command("git", "status")
	mainStatusCmd.Dir = repoPath
	mainStatusOutput, err := mainStatusCmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to check main repository status: %w", err)
	}

	fmt.Printf("Main repository status after submodule updates:\n%s\n", string(mainStatusOutput))

	return anyUpdated, nil
}

// updateSubmodules updates all submodules in the main repository
func (m *Manager) updateSubmodules() error {
	repoPath := filepath.Join(m.config.WorkingDir, m.config.MainRepo.GetDirectory())

	// Check if .gitmodules exists
	gitmodulesPath := filepath.Join(repoPath, ".gitmodules")
	if _, err := os.Stat(gitmodulesPath); os.IsNotExist(err) {
		return fmt.Errorf(".gitmodules file not found in main repository")
	}

	// Initialize submodules if they haven't been initialized yet
	initCmd := exec.Command("git", "submodule", "init")
	initCmd.Dir = repoPath
	if output, err := initCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git submodule init failed: %w, output: %s", err, string(output))
	}

	// Update submodules
	updateCmd := exec.Command("git", "submodule", "update", "--recursive", "--remote")
	updateCmd.Dir = repoPath

	// Set environment variables for authentication if using SSH
	if m.config.MainRepo.GetAuth().Type == "ssh" && m.config.MainRepo.GetAuth().SSHKeyPath != "" {
		updateCmd.Env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no", m.config.MainRepo.GetAuth().SSHKeyPath))
	}

	if output, err := updateCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git submodule update failed: %w, output: %s", err, string(output))
	}

	// Get list of submodules for logging
	submodules, err := m.listSubmodules()
	if err != nil {
		fmt.Printf("Warning: Could not list submodules: %v\n", err)
	} else {
		fmt.Printf("Updated %d submodules: %s\n", len(submodules), strings.Join(submodules, ", "))
	}

	return nil
}

// commitSubmoduleChangesToMainRepo commits submodule changes to the main repository
func (m *Manager) commitSubmoduleChangesToMainRepo(branch string) error {
	repoPath := filepath.Join(m.config.WorkingDir, m.config.MainRepo.GetDirectory())

	// Configure Git user for the commit if provided
	if m.config.CommitConfig.UserName != "" {
		configNameCmd := exec.Command("git", "config", "user.name", m.config.CommitConfig.UserName)
		configNameCmd.Dir = repoPath
		if output, err := configNameCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to set git user.name: %w, output: %s", err, string(output))
		}
	}

	if m.config.CommitConfig.UserEmail != "" {
		configEmailCmd := exec.Command("git", "config", "user.email", m.config.CommitConfig.UserEmail)
		configEmailCmd.Dir = repoPath
		if output, err := configEmailCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to set git user.email: %w, output: %s", err, string(output))
		}
	}

	// Check if there are changes to submodules
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = repoPath
	output, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	statusOutput := strings.TrimSpace(string(output))
	if len(statusOutput) == 0 {
		fmt.Println("No changes to commit in main repository")
		return nil
	}

	fmt.Printf("Changes detected in main repository:\n%s\n", statusOutput)

	// Get the list of modified submodules
	submodules, err := m.listSubmodules()
	if err != nil {
		return fmt.Errorf("failed to list submodules: %w", err)
	}

	// Add all submodule changes
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = repoPath
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %w, output: %s", err, string(output))
	}

	// Create the commit message with timestamp and submodule details
	timestamp := time.Now().Format(time.RFC3339)
	commitMessage := m.config.CommitConfig.Message
	if commitMessage == "" {
		commitMessage = "Update submodules [Git Watcher Auto-Commit]"
	}

	// Get the current commit hashes of all submodules
	submoduleDetails := make([]string, 0, len(submodules))
	for _, submodule := range submodules {
		hashCmd := exec.Command("git", "rev-parse", "HEAD")
		hashCmd.Dir = filepath.Join(repoPath, submodule)
		if hashOutput, err := hashCmd.Output(); err == nil {
			submoduleDetails = append(submoduleDetails, fmt.Sprintf("%s: %s", submodule, strings.TrimSpace(string(hashOutput))))
		}
	}

	// Format the commit message
	commitMessage = fmt.Sprintf("%s\n\nBranch: %s\nTimestamp: %s\n\nUpdated submodules:\n%s",
		commitMessage,
		branch,
		timestamp,
		strings.Join(submoduleDetails, "\n"))

	// Commit the changes
	commitCmd := exec.Command("git", "commit", "-m", commitMessage)
	commitCmd.Dir = repoPath
	if output, err := commitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %w, output: %s", err, string(output))
	}

	fmt.Printf("Committed submodule changes to main repository on branch %s\n", branch)
	fmt.Printf("Commit message:\n%s\n", commitMessage)

	// Push the changes if authentication is configured
	if m.config.MainRepo.GetAuth().Type != "none" {
		pushCmd := exec.Command("git", "push", "origin", branch)
		pushCmd.Dir = repoPath
		m.setupCredentials(m.config.MainRepo, pushCmd)

		if output, err := pushCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git push failed: %w, output: %s", err, string(output))
		}

		fmt.Printf("Pushed submodule changes to remote repository on branch %s\n", branch)
	}

	return nil
}

// listSubmodules returns a list of submodule names from .gitmodules
func (m *Manager) listSubmodules() ([]string, error) {
	repoPath := filepath.Join(m.config.WorkingDir, m.config.MainRepo.GetDirectory())
	gitmodulesPath := filepath.Join(repoPath, ".gitmodules")

	data, err := os.ReadFile(gitmodulesPath)
	if err != nil {
		return nil, err
	}

	content := string(data)
	submoduleRegex := regexp.MustCompile(`\[submodule "([^"]+)"\]`)
	matches := submoduleRegex.FindAllStringSubmatch(content, -1)

	submodules := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) >= 2 {
			submodules = append(submodules, match[1])
		}
	}

	return submodules, nil
}

// checkAndUpdateRepo checks if a repository has new commits and updates it if necessary
// Returns true if the repository was updated
func (m *Manager) checkAndUpdateRepo(repo config.RepositoryInterface) (bool, error) {
	repoPath := filepath.Join(m.config.WorkingDir, repo.GetDirectory())

	// Check if the repository exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		// Clone the repository if it doesn't exist
		if err := m.cloneRepo(repo); err != nil {
			return false, err
		}
		return true, nil
	}

	// Check for new commits
	hasNewCommits, err := m.hasNewCommits(repo)
	if err != nil {
		return false, err
	}

	// If there are new commits, pull the changes
	if hasNewCommits {
		if err := m.pullRepo(repo); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

// setupCredentials configures Git credentials based on authentication settings
func (m *Manager) setupCredentials(repo config.RepositoryInterface, cmd *exec.Cmd) {
	switch repo.GetAuth().Type {
	case "basic":
		if repo.GetAuth().Username != "" && repo.GetAuth().Password != "" {
			// For HTTPS with basic auth, use credential helper
			cmd.Env = append(os.Environ(),
				"GIT_ASKPASS=echo",
				fmt.Sprintf("GIT_USERNAME=%s", repo.GetAuth().Username),
				fmt.Sprintf("GIT_PASSWORD=%s", repo.GetAuth().Password),
			)
		}
	case "ssh":
		if repo.GetAuth().SSHKeyPath != "" {
			// For SSH authentication with a key file
			cmd.Env = append(os.Environ(),
				fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no", repo.GetAuth().SSHKeyPath),
			)
		} else if repo.GetAuth().SSHPrivateKey != "" {
			// If SSH key is provided as a string, write it to a temporary file
			tmpDir, err := os.MkdirTemp("", "git-ssh-key")
			if err == nil {
				keyPath := filepath.Join(tmpDir, "id_rsa")
				if err := os.WriteFile(keyPath, []byte(repo.GetAuth().SSHPrivateKey), 0600); err == nil {
					cmd.Env = append(os.Environ(),
						fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no", keyPath),
					)
					// Clean up the temporary file when the command finishes
					defer os.RemoveAll(tmpDir)
				}
			}
		}
	}
}

// cloneRepo clones a Git repository
func (m *Manager) cloneRepo(repo config.RepositoryInterface) error {
	repoPath := filepath.Join(m.config.WorkingDir, repo.GetDirectory())

	args := []string{"clone", "--branch", repo.GetBranch()}

	// Handle authentication for SSH URLs
	if strings.HasPrefix(repo.GetURL(), "git@") && repo.GetAuth().Type == "ssh" {
		// SSH URL, will use SSH key based authentication
		args = append(args, repo.GetURL(), repoPath)
		cmd := exec.Command("git", args...)
		m.setupCredentials(repo, cmd)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git clone failed: %w, output: %s", err, string(output))
		}
	} else {
		// HTTP(S) URL
		if repo.GetAuth().Type == "basic" && repo.GetAuth().Username != "" && repo.GetAuth().Password != "" {
			// Rewrite URL to include basic auth
			urlParts := strings.SplitN(repo.GetURL(), "://", 2)
			if len(urlParts) == 2 {
				repoURL := fmt.Sprintf("%s://%s:%s@%s",
					urlParts[0],
					repo.GetAuth().Username,
					repo.GetAuth().Password,
					urlParts[1],
				)
				args = append(args, repoURL, repoPath)
			}
		} else {
			args = append(args, repo.GetURL(), repoPath)
		}
		cmd := exec.Command("git", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git clone failed: %w, output: %s", err, string(output))
		}
	}

	fmt.Printf("Cloned repository %s branch %s to %s\n", repo.GetURL(), repo.GetBranch(), repoPath)

	// If this is the main repository and we're using submodules, initialize them
	if m.config.UseSubmodules && repo.GetDirectory() == m.config.MainRepo.GetDirectory() {
		if err := m.updateSubmodules(); err != nil {
			fmt.Printf("Warning: Failed to initialize submodules: %v\n", err)
		}
	}

	return nil
}

// hasNewCommits checks if a repository has new commits
func (m *Manager) hasNewCommits(repo config.RepositoryInterface) (bool, error) {
	repoPath := filepath.Join(m.config.WorkingDir, repo.GetDirectory())

	// Fetch updates from the remote repository
	fetchCmd := exec.Command("git", "fetch", "origin", repo.GetBranch())
	fetchCmd.Dir = repoPath
	m.setupCredentials(repo, fetchCmd)

	if output, err := fetchCmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("git fetch failed: %w, output: %s", err, string(output))
	}

	// Check if the local branch is behind the remote branch
	diffCmd := exec.Command("git", "rev-list", "HEAD..origin/"+repo.GetBranch(), "--count")
	diffCmd.Dir = repoPath
	output, err := diffCmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check for new commits: %w", err)
	}

	count := strings.TrimSpace(string(output))
	return count != "0", nil
}

// pullRepo pulls the latest changes from a repository
func (m *Manager) pullRepo(repo config.RepositoryInterface) error {
	repoPath := filepath.Join(m.config.WorkingDir, repo.GetDirectory())

	// Make sure we're on the right branch
	checkoutCmd := exec.Command("git", "checkout", repo.GetBranch())
	checkoutCmd.Dir = repoPath
	if output, err := checkoutCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout failed: %w, output: %s", err, string(output))
	}

	// Pull the changes
	pullCmd := exec.Command("git", "pull", "origin", repo.GetBranch())
	pullCmd.Dir = repoPath
	m.setupCredentials(repo, pullCmd)

	if output, err := pullCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git pull failed: %w, output: %s", err, string(output))
	}

	fmt.Printf("Updated repository %s branch %s at %s\n", repo.GetURL(), repo.GetBranch(), repoPath)
	return nil
}

// GetLastCommitHash returns the last commit hash of a repository
func (m *Manager) GetLastCommitHash(repo config.RepositoryInterface) (string, error) {
	repoPath := filepath.Join(m.config.WorkingDir, repo.GetDirectory())

	// Check if the repository exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return "", errors.New("repository does not exist")
	}

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get last commit hash: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetConfig returns the Git configuration
func (m *Manager) GetConfig() *config.GitConfig {
	return m.config
}

// mergeCommitConfig 合并主仓库和制品仓库的提交配置，制品仓库的配置优先级更高
func (m *Manager) mergeCommitConfig(mainConfig, artifactsConfig config.CommitConfig) config.CommitConfig {
	merged := mainConfig // 从主仓库配置开始

	// 如果制品仓库配置中有值，则覆盖主仓库的配置
	if artifactsConfig.UserName != "" {
		merged.UserName = artifactsConfig.UserName
	}
	if artifactsConfig.UserEmail != "" {
		merged.UserEmail = artifactsConfig.UserEmail
	}
	if artifactsConfig.Message != "" {
		merged.Message = artifactsConfig.Message
	}

	return merged
}

// UpdateArtifactsRepo 更新制品仓库
func (m *Manager) UpdateArtifactsRepo(repoName, pkgName, version string) error {
	// 获取 Git 操作锁
	m.gitOpLock.Lock()
	defer m.gitOpLock.Unlock()

	// 检查制品仓库是否配置
	if m.config.ArtifactsRepo == nil {
		return fmt.Errorf("artifacts repository is not configured")
	}

	// 检查仓库是否存在
	repoPath := filepath.Join(m.config.WorkingDir, m.config.ArtifactsRepo.Directory)
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		// 如果仓库不存在，则克隆
		if err := m.cloneRepo(m.config.ArtifactsRepo); err != nil {
			return fmt.Errorf("failed to clone artifacts repository: %w", err)
		}
	}

	// 创建或切换到 feature 分支
	featureBranch := fmt.Sprintf("feature-%s", repoName)

	// 检查远程分支是否存在
	lsRemoteCmd := exec.Command("git", "ls-remote", "--heads", "origin", featureBranch)
	lsRemoteCmd.Dir = repoPath
	m.setupCredentials(m.config.ArtifactsRepo, lsRemoteCmd)
	lsRemoteOutput, err := lsRemoteCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to check remote branch: %w, output: %s", err, string(lsRemoteOutput))
	}

	// 如果远程分支存在，则拉取
	if len(strings.TrimSpace(string(lsRemoteOutput))) > 0 {
		// 切换到 feature 分支
		checkoutCmd := exec.Command("git", "checkout", featureBranch)
		checkoutCmd.Dir = repoPath
		if _, err := checkoutCmd.CombinedOutput(); err != nil {
			// 如果本地分支不存在，则创建并拉取
			checkoutCmd = exec.Command("git", "checkout", "-b", featureBranch, "origin/"+featureBranch)
			checkoutCmd.Dir = repoPath
			if output, err := checkoutCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to checkout feature branch: %w, output: %s", err, string(output))
			}
		}
	} else {
		// 如果远程分支不存在，则创建新分支
		checkoutCmd := exec.Command("git", "checkout", "-b", featureBranch)
		checkoutCmd.Dir = repoPath
		if output, err := checkoutCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create feature branch: %w, output: %s", err, string(output))
		}
	}

	// 创建或更新 {repoName}.jsonnet 文件
	jsonnetPath := filepath.Join(repoPath, fmt.Sprintf("%s.jsonnet", repoName))

	// 获取文件锁
	fileLock := m.getFileLock(jsonnetPath)
	fileLock.Lock()
	defer fileLock.Unlock()

	// 从 version 中提取包名（去掉最后一段版本号）
	parts := strings.Split(version, "-")
	versionPrefix := strings.Join(parts[:len(parts)-1], "-")

	// 读取现有内容（如果文件存在）
	var existingContent map[string]interface{}
	if data, err := os.ReadFile(jsonnetPath); err == nil {
		if err := json.Unmarshal(data, &existingContent); err != nil {
			return fmt.Errorf("failed to parse existing jsonnet file: %w", err)
		}
	} else {
		existingContent = make(map[string]interface{})
	}

	// 更新或创建内容
	repoContent, ok := existingContent[repoName].(map[string]interface{})
	if !ok {
		repoContent = make(map[string]interface{})
		existingContent[repoName] = repoContent
	}

	pkgContent, ok := repoContent[pkgName].(map[string]interface{})
	if !ok {
		pkgContent = make(map[string]interface{})
		repoContent[pkgName] = pkgContent
	}

	// 检查版本是否已存在且相同
	if existingVersion, exists := pkgContent[versionPrefix]; exists && existingVersion == version {
		fmt.Printf("Version %s already exists and is up to date\n", version)
		return nil
	}

	pkgContent[versionPrefix] = version

	// 将更新后的内容写入文件
	jsonnetContent, err := json.MarshalIndent(existingContent, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal jsonnet content: %w", err)
	}

	if err := os.WriteFile(jsonnetPath, jsonnetContent, 0644); err != nil {
		return fmt.Errorf("failed to write jsonnet file: %w", err)
	}

	// 添加文件到暂存区
	addCmd := exec.Command("git", "add", fmt.Sprintf("%s.jsonnet", repoName))
	addCmd.Dir = repoPath
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %w, output: %s", err, string(output))
	}

	// 检查是否有更改需要提交
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = repoPath
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	// 只有在有更改时才提交
	if len(strings.TrimSpace(string(statusOutput))) > 0 {
		// 合并提交配置
		commitConfig := m.mergeCommitConfig(m.config.CommitConfig, m.config.ArtifactsRepo.CommitConfig)

		// 设置 Git 用户信息
		if commitConfig.UserName != "" {
			configNameCmd := exec.Command("git", "config", "user.name", commitConfig.UserName)
			configNameCmd.Dir = repoPath
			if output, err := configNameCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to set git user.name: %w, output: %s", err, string(output))
			}
		}
		if commitConfig.UserEmail != "" {
			configEmailCmd := exec.Command("git", "config", "user.email", commitConfig.UserEmail)
			configEmailCmd.Dir = repoPath
			if output, err := configEmailCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to set git user.email: %w, output: %s", err, string(output))
			}
		}

		// 提交更改
		commitMessage := commitConfig.Message
		if commitMessage == "" {
			commitMessage = "Update artifacts"
		}
		timestamp := time.Now().Format(time.RFC3339)
		commitMessage = fmt.Sprintf("%s\n\nTime: %s\nArtifactRepoName: %s\nPackage: %s\nVersion: %s",
			commitMessage,
			timestamp,
			repoName,
			pkgName,
			version,
		)

		commitCmd := exec.Command("git", "commit", "-m", commitMessage)
		commitCmd.Dir = repoPath
		if output, err := commitCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git commit failed: %w, output: %s", err, string(output))
		}

		// 强制推送到远程仓库
		pushCmd := exec.Command("git", "push", "-f", "origin", featureBranch)
		pushCmd.Dir = repoPath
		m.setupCredentials(m.config.ArtifactsRepo, pushCmd)
		if output, err := pushCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git push failed: %w, output: %s", err, string(output))
		}

		// 合并到指定分支
		targetBranch := m.config.ArtifactsRepo.AutoBranchName
		if targetBranch == "" {
			targetBranch = m.config.ArtifactsRepo.Branch // 如果未配置 autoBranchName，则使用默认分支
		}

		// 切换到目标分支
		checkoutTargetCmd := exec.Command("git", "checkout", targetBranch)
		checkoutTargetCmd.Dir = repoPath
		if output, err := checkoutTargetCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to checkout target branch %s: %w, output: %s", targetBranch, err, string(output))
		}

		// 清理未合并的文件
		cleanupCmd := exec.Command("git", "reset", "--hard", "HEAD")
		cleanupCmd.Dir = repoPath
		if output, err := cleanupCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to cleanup unmerged files: %w, output: %s", err, string(output))
		}

		// 拉取目标分支最新代码
		pullTargetCmd := exec.Command("git", "pull", "origin", targetBranch)
		pullTargetCmd.Dir = repoPath
		m.setupCredentials(m.config.ArtifactsRepo, pullTargetCmd)
		if output, err := pullTargetCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to pull target branch %s: %w, output: %s", targetBranch, err, string(output))
		}

		// 合并 feature 分支
		mergeCmd := exec.Command("git", "merge", "--no-ff", "--strategy-option=theirs", featureBranch)
		mergeCmd.Dir = repoPath
		if _, err := mergeCmd.CombinedOutput(); err != nil {
			// 如果合并失败，尝试使用 --strategy-option=theirs 选项
			mergeCmd = exec.Command("git", "merge", "--no-ff", "--strategy-option=theirs", featureBranch)
			mergeCmd.Dir = repoPath
			if output, err := mergeCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to merge branch %s into %s: %w, output: %s", featureBranch, targetBranch, err, string(output))
			}
		}

		// 推送合并后的更改到远程
		pushMergeCmd := exec.Command("git", "push", "origin", targetBranch)
		pushMergeCmd.Dir = repoPath
		m.setupCredentials(m.config.ArtifactsRepo, pushMergeCmd)
		if output, err := pushMergeCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to push merged changes to %s: %w, output: %s", targetBranch, err, string(output))
		}

		fmt.Printf("Successfully merged branch %s into %s and pushed to remote\n", featureBranch, targetBranch)
	} else {
		fmt.Printf("No changes detected in %s.jsonnet, skipping commit and merge\n", repoName)
	}

	return nil
}
