package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Environment variable names
const (
	// Server
	EnvServerPort = "GIT_WATCHER_SERVER_PORT"

	// Git
	EnvGitMainRepoURL       = "GIT_WATCHER_MAIN_REPO_URL"
	EnvGitMainRepoBranch    = "GIT_WATCHER_MAIN_REPO_BRANCH"
	EnvGitMainRepoDirectory = "GIT_WATCHER_MAIN_REPO_DIRECTORY"
	EnvGitWorkingDir        = "GIT_WATCHER_WORKING_DIR"
	EnvGitUseSubmodules     = "GIT_WATCHER_USE_SUBMODULES"
	EnvGitBranches          = "GIT_WATCHER_BRANCHES"
	EnvGitAutoCommit        = "GIT_WATCHER_AUTO_COMMIT"
	EnvGitCommitUserName    = "GIT_WATCHER_COMMIT_USER_NAME"
	EnvGitCommitUserEmail   = "GIT_WATCHER_COMMIT_USER_EMAIL"
	EnvGitCommitMessage     = "GIT_WATCHER_COMMIT_MESSAGE"

	// Artifacts Repo
	EnvGitArtifactsRepoURL       = "GIT_WATCHER_ARTIFACTS_REPO_URL"
	EnvGitArtifactsRepoBranch    = "GIT_WATCHER_ARTIFACTS_REPO_BRANCH"
	EnvGitArtifactsRepoDirectory = "GIT_WATCHER_ARTIFACTS_REPO_DIRECTORY"
	EnvGitArtifactsAuthType      = "GIT_WATCHER_ARTIFACTS_AUTH_TYPE"
	EnvGitArtifactsAuthUsername  = "GIT_WATCHER_ARTIFACTS_AUTH_USERNAME"
	EnvGitArtifactsAuthPassword  = "GIT_WATCHER_ARTIFACTS_AUTH_PASSWORD"
	EnvGitArtifactsAuthSSHKey    = "GIT_WATCHER_ARTIFACTS_AUTH_SSH_KEY_PATH"
	EnvGitArtifactsAuthSSHPriv   = "GIT_WATCHER_ARTIFACTS_AUTH_SSH_PRIVATE_KEY"

	// Auth
	EnvGitAuthType          = "GIT_WATCHER_AUTH_TYPE"
	EnvGitAuthUsername      = "GIT_WATCHER_AUTH_USERNAME"
	EnvGitAuthPassword      = "GIT_WATCHER_AUTH_PASSWORD"
	EnvGitAuthSSHKeyPath    = "GIT_WATCHER_AUTH_SSH_KEY_PATH"
	EnvGitAuthSSHPrivateKey = "GIT_WATCHER_AUTH_SSH_PRIVATE_KEY"

	// Webhook
	EnvWebhookCallbackURL = "GIT_WATCHER_WEBHOOK_CALLBACK_URL"
	EnvWebhookSecret      = "GIT_WATCHER_WEBHOOK_SECRET"
	EnvWebhookMethod      = "GIT_WATCHER_WEBHOOK_METHOD"

	// Schedule
	EnvScheduleCheckInterval = "GIT_WATCHER_CHECK_INTERVAL"

	// Artifacts Repo
	EnvArtifactsRepoURL            = "ARTIFACTS_REPO_URL"
	EnvArtifactsRepoBranch         = "ARTIFACTS_REPO_BRANCH"
	EnvArtifactsRepoDirectory      = "ARTIFACTS_REPO_DIRECTORY"
	EnvArtifactsRepoUsername       = "ARTIFACTS_REPO_USERNAME"
	EnvArtifactsRepoPassword       = "ARTIFACTS_REPO_PASSWORD"
	EnvArtifactsRepoUseMainAuth    = "ARTIFACTS_REPO_USE_MAIN_AUTH"
	EnvArtifactsRepoUseMainCommit  = "ARTIFACTS_REPO_USE_MAIN_COMMIT"
	EnvArtifactsRepoCommitUsername = "ARTIFACTS_REPO_COMMIT_USERNAME"
	EnvArtifactsRepoCommitEmail    = "ARTIFACTS_REPO_COMMIT_EMAIL"
)

// RepositoryInterface 仓库接口
type RepositoryInterface interface {
	GetURL() string
	GetBranch() string
	GetDirectory() string
	GetAuth() AuthConfig
	GetCommitConfig() CommitConfig
}

// Repository 仓库配置
type Repository struct {
	URL          string       `json:"url"`          // 仓库URL
	Branch       string       `json:"branch"`       // 分支名称
	Directory    string       `json:"directory"`    // 本地目录
	Auth         AuthConfig   `json:"auth"`         // 认证配置
	CommitConfig CommitConfig `json:"commitConfig"` // 提交信息配置
}

// GetURL 实现 RepositoryInterface 接口
func (r *Repository) GetURL() string {
	return r.URL
}

// GetBranch 实现 RepositoryInterface 接口
func (r *Repository) GetBranch() string {
	return r.Branch
}

// GetDirectory 实现 RepositoryInterface 接口
func (r *Repository) GetDirectory() string {
	return r.Directory
}

// GetAuth 实现 RepositoryInterface 接口
func (r *Repository) GetAuth() AuthConfig {
	return r.Auth
}

// GetCommitConfig 实现 RepositoryInterface 接口
func (r *Repository) GetCommitConfig() CommitConfig {
	return r.CommitConfig
}

// ArtifactsRepo 制品仓库配置
type ArtifactsRepo struct {
	URL            string       `json:"url"`            // 仓库URL
	Branch         string       `json:"branch"`         // 分支名称
	Directory      string       `json:"directory"`      // 本地目录
	Auth           AuthConfig   `json:"auth"`           // 认证配置
	UseMainAuth    bool         `json:"useMainAuth"`    // 是否使用主仓库的认证信息
	UseMainCommit  bool         `json:"useMainCommit"`  // 是否使用主仓库的提交信息
	CommitConfig   CommitConfig `json:"commitConfig"`   // 提交信息配置
	AutoBranchName string       `json:"autoBranchName"` // 自动合并的目标分支名称
}

// GetURL 实现 RepositoryInterface 接口
func (r *ArtifactsRepo) GetURL() string {
	return r.URL
}

// GetBranch 实现 RepositoryInterface 接口
func (r *ArtifactsRepo) GetBranch() string {
	return r.Branch
}

// GetDirectory 实现 RepositoryInterface 接口
func (r *ArtifactsRepo) GetDirectory() string {
	return r.Directory
}

// GetAuth 实现 RepositoryInterface 接口
func (r *ArtifactsRepo) GetAuth() AuthConfig {
	return r.Auth
}

// GetCommitConfig 实现 RepositoryInterface 接口
func (r *ArtifactsRepo) GetCommitConfig() CommitConfig {
	return r.CommitConfig
}

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `json:"server"`
	Git      GitConfig      `json:"git"`
	Webhook  WebhookConfig  `json:"webhook"`
	Schedule ScheduleConfig `json:"schedule"`
	// 添加制品仓库配置
	ArtifactsRepo ArtifactsRepo `json:"artifactsRepo"`
}

// ServerConfig contains server-specific configuration
type ServerConfig struct {
	Port int `json:"port"`
}

// GitConfig contains Git-related configuration
type GitConfig struct {
	WorkingDir    string         `json:"workingDir"`    // 工作目录
	UseSubmodules bool           `json:"useSubmodules"` // 是否使用子模块
	Branches      []string       `json:"branches"`      // 分支列表
	AutoCommit    bool           `json:"autoCommit"`    // 是否自动提交
	CommitConfig  CommitConfig   `json:"commitConfig"`  // 提交信息配置
	MainRepo      *Repository    `json:"mainRepo"`      // 主仓库配置
	ArtifactsRepo *ArtifactsRepo `json:"artifactsRepo"` // 制品仓库配置
}

// CommitConfig 提交信息配置
type CommitConfig struct {
	UserName  string `json:"userName"`  // Git 用户名
	UserEmail string `json:"userEmail"` // Git 邮箱
	Message   string `json:"message"`   // Git 提交信息
}

// AuthConfig represents authentication configuration for Git
type AuthConfig struct {
	Type          string `json:"type"` // "none", "basic", "ssh"
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	SSHPrivateKey string `json:"sshPrivateKey,omitempty"`
	SSHKeyPath    string `json:"sshKeyPath,omitempty"`
}

// WebhookConfig contains webhook-related configuration
type WebhookConfig struct {
	CallbackURL string `json:"callbackUrl"`
	Secret      string `json:"secret"`
	Method      string `json:"method"`
}

// ScheduleConfig contains scheduling configuration
type ScheduleConfig struct {
	CheckInterval time.Duration `json:"-"` // 使用自定义解析
	RawInterval   interface{}   `json:"checkInterval"`
}

// Custom UnmarshalJSON for ScheduleConfig
func (sc *ScheduleConfig) UnmarshalJSON(data []byte) error {
	type Alias ScheduleConfig
	aux := struct {
		*Alias
	}{
		Alias: (*Alias)(sc),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// 处理 RawInterval
	switch v := sc.RawInterval.(type) {
	case float64: // JSON 中的数字默认会被解析为 float64
		sc.CheckInterval = time.Duration(v)
	case string:
		// 尝试解析为时间字符串
		duration, err := time.ParseDuration(v)
		if err != nil {
			// 尝试解析为整数字符串（纳秒）
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				sc.CheckInterval = time.Duration(i)
			} else {
				return fmt.Errorf("invalid check interval: %v", v)
			}
		} else {
			sc.CheckInterval = duration
		}
	default:
		sc.CheckInterval = 10 * time.Minute // 默认值
	}

	return nil
}

// MarshalJSON 自定义 JSON 序列化
func (sc ScheduleConfig) MarshalJSON() ([]byte, error) {
	// 创建新的结构体，避免递归调用 MarshalJSON
	return json.Marshal(struct {
		CheckInterval interface{} `json:"checkInterval"`
	}{
		CheckInterval: sc.CheckInterval.String(), // 序列化为字符串形式
	})
}

// LoadConfig loads the configuration from a file
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 设置默认值
	if config.Git.MainRepo.Branch == "" {
		config.Git.MainRepo.Branch = "main"
	}

	// 使用环境变量覆盖配置
	OverrideWithEnv(&config)

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// OverrideWithEnv overrides configuration values with environment variables
func OverrideWithEnv(config *Config) {
	// Server config
	if port, exists := getEnvInt(EnvServerPort); exists {
		config.Server.Port = port
	}

	// Git Main Repo config
	if url := os.Getenv(EnvGitMainRepoURL); url != "" {
		config.Git.MainRepo.URL = url
	}
	if branch := os.Getenv(EnvGitMainRepoBranch); branch != "" {
		config.Git.MainRepo.Branch = branch
	}
	if dir := os.Getenv(EnvGitMainRepoDirectory); dir != "" {
		config.Git.MainRepo.Directory = dir
	}
	if workingDir := os.Getenv(EnvGitWorkingDir); workingDir != "" {
		config.Git.WorkingDir = workingDir
	}
	if useSubmodules, exists := getEnvBool(EnvGitUseSubmodules); exists {
		config.Git.UseSubmodules = useSubmodules
	}
	if branches := os.Getenv(EnvGitBranches); branches != "" {
		config.Git.Branches = strings.Split(branches, ",")
	}

	// Git Auth config
	if authType := os.Getenv(EnvGitAuthType); authType != "" {
		config.Git.MainRepo.Auth.Type = authType
	}
	if username := os.Getenv(EnvGitAuthUsername); username != "" {
		config.Git.MainRepo.Auth.Username = username
	}
	if password := os.Getenv(EnvGitAuthPassword); password != "" {
		config.Git.MainRepo.Auth.Password = password
	}
	if sshKeyPath := os.Getenv(EnvGitAuthSSHKeyPath); sshKeyPath != "" {
		config.Git.MainRepo.Auth.SSHKeyPath = sshKeyPath
	}
	if sshPrivateKey := os.Getenv(EnvGitAuthSSHPrivateKey); sshPrivateKey != "" {
		config.Git.MainRepo.Auth.SSHPrivateKey = sshPrivateKey
	}

	// Webhook config
	if callbackURL := os.Getenv(EnvWebhookCallbackURL); callbackURL != "" {
		config.Webhook.CallbackURL = callbackURL
	}
	if secret := os.Getenv(EnvWebhookSecret); secret != "" {
		config.Webhook.Secret = secret
	}
	if method := os.Getenv(EnvWebhookMethod); method != "" {
		config.Webhook.Method = method
	}

	// Schedule config
	if interval, exists := getEnvDuration(EnvScheduleCheckInterval); exists {
		config.Schedule.CheckInterval = interval
	}

	// Git auto commit config
	if autoCommit, exists := getEnvBool(EnvGitAutoCommit); exists {
		config.Git.AutoCommit = autoCommit
	}
	if userName := os.Getenv(EnvGitCommitUserName); userName != "" {
		config.Git.CommitConfig.UserName = userName
	}
	if userEmail := os.Getenv(EnvGitCommitUserEmail); userEmail != "" {
		config.Git.CommitConfig.UserEmail = userEmail
	}
	if commitMessage := os.Getenv(EnvGitCommitMessage); commitMessage != "" {
		config.Git.CommitConfig.Message = commitMessage
	}

	// Artifacts Repo config
	if url := os.Getenv(EnvGitArtifactsRepoURL); url != "" {
		config.Git.ArtifactsRepo.URL = url
	}
	if branch := os.Getenv(EnvGitArtifactsRepoBranch); branch != "" {
		config.Git.ArtifactsRepo.Branch = branch
	}
	if dir := os.Getenv(EnvGitArtifactsRepoDirectory); dir != "" {
		config.Git.ArtifactsRepo.Directory = dir
	}
	if useMainAuth, exists := getEnvBool("GIT_WATCHER_ARTIFACTS_USE_MAIN_AUTH"); exists {
		config.Git.ArtifactsRepo.UseMainAuth = useMainAuth
	}
	if useMainCommit, exists := getEnvBool("GIT_WATCHER_ARTIFACTS_USE_MAIN_COMMIT"); exists {
		config.Git.ArtifactsRepo.UseMainCommit = useMainCommit
	}

	// 只有在不使用主仓库认证时才设置制品仓库的认证信息
	if !config.Git.ArtifactsRepo.UseMainAuth {
		if authType := os.Getenv(EnvGitArtifactsAuthType); authType != "" {
			config.Git.ArtifactsRepo.Auth.Type = authType
		}
		if username := os.Getenv(EnvGitArtifactsAuthUsername); username != "" {
			config.Git.ArtifactsRepo.Auth.Username = username
		}
		if password := os.Getenv(EnvGitArtifactsAuthPassword); password != "" {
			config.Git.ArtifactsRepo.Auth.Password = password
		}
		if sshKeyPath := os.Getenv(EnvGitArtifactsAuthSSHKey); sshKeyPath != "" {
			config.Git.ArtifactsRepo.Auth.SSHKeyPath = sshKeyPath
		}
		if sshPrivateKey := os.Getenv(EnvGitArtifactsAuthSSHPriv); sshPrivateKey != "" {
			config.Git.ArtifactsRepo.Auth.SSHPrivateKey = sshPrivateKey
		}
	}
}

// getEnvInt gets an integer from an environment variable
func getEnvInt(key string) (int, bool) {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue, true
		}
	}
	return 0, false
}

// getEnvBool gets a boolean from an environment variable
func getEnvBool(key string) (bool, bool) {
	if value := os.Getenv(key); value != "" {
		lowValue := strings.ToLower(value)
		if lowValue == "true" || lowValue == "1" || lowValue == "yes" {
			return true, true
		} else if lowValue == "false" || lowValue == "0" || lowValue == "no" {
			return false, true
		}
	}
	return false, false
}

// getEnvDuration gets a duration from an environment variable
func getEnvDuration(key string) (time.Duration, bool) {
	if value := os.Getenv(key); value != "" {
		// Try parsing as nanoseconds
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return time.Duration(intValue), true
		}

		// Try parsing as duration string
		if duration, err := time.ParseDuration(value); err == nil {
			return duration, true
		}
	}
	return 0, false
}

// validateConfig validates the configuration values
func validateConfig(config *Config) error {
	if config.Server.Port <= 0 {
		return fmt.Errorf("server port must be greater than zero")
	}

	// Validate main repository configuration
	if config.Git.MainRepo.URL == "" {
		return fmt.Errorf("main repository URL is required")
	}
	if config.Git.MainRepo.Branch == "" {
		return fmt.Errorf("main repository branch is required")
	}
	if config.Git.MainRepo.Directory == "" {
		return fmt.Errorf("main repository directory is required")
	}

	// Validate artifacts repository configuration
	if config.Git.ArtifactsRepo.URL == "" {
		return fmt.Errorf("artifacts repository URL is required")
	}
	if config.Git.ArtifactsRepo.Branch == "" {
		return fmt.Errorf("artifacts repository branch is required")
	}
	if config.Git.ArtifactsRepo.Directory == "" {
		return fmt.Errorf("artifacts repository directory is required")
	}

	// Validate webhook configuration
	if config.Webhook.CallbackURL == "" {
		return fmt.Errorf("webhook callback URL is required")
	}

	// Validate schedule configuration
	if config.Schedule.CheckInterval <= 0 {
		return fmt.Errorf("check interval must be greater than zero")
	}

	return nil
}

// SaveConfig saves the configuration to the specified file
func SaveConfig(config *Config, filename string) error {
	// 确保在保存前正确设置 RawInterval
	if config.Schedule.RawInterval == nil {
		config.Schedule.RawInterval = config.Schedule.CheckInterval.String()
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
