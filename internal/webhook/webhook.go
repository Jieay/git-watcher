package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	config "github.com/Jieay/git-watcher/configs"
)

// Client handles webhook operations
type Client struct {
	config *config.WebhookConfig
	client *http.Client
}

// NewClient creates a new webhook client
func NewClient(cfg *config.WebhookConfig) *Client {
	return &Client{
		config: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// WebhookPayload represents the payload to be sent to the webhook
type WebhookPayload struct {
	Event       string                `json:"event"`
	Timestamp   time.Time             `json:"timestamp"`
	Branch      string                `json:"branch,omitempty"` // Branch that was updated
	RepoUpdates map[string]RepoUpdate `json:"repoUpdates"`
	Message     string                `json:"message"`
}

// RepoUpdate contains information about a repository update
type RepoUpdate struct {
	Repository string    `json:"repository"`
	Branch     string    `json:"branch"`
	Timestamp  time.Time `json:"timestamp"`
	CommitHash string    `json:"commitHash"`
}

// WebhookTriggerRequest represents the payload received from an external webhook trigger
type WebhookTriggerRequest struct {
	Event     string `json:"event"`
	Branch    string `json:"branch"`    // Optional branch to check
	Reference string `json:"reference"` // Git reference (alternative to branch)
}

// SendNotification sends a webhook notification about repository updates
func (c *Client) SendNotification(payload WebhookPayload) error {
	if c.config.CallbackURL == "" {
		return fmt.Errorf("webhook URL is not configured")
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// 使用配置的请求方法，默认为 POST
	method := c.config.Method
	if method == "" {
		method = "POST"
	}

	req, err := http.NewRequest(method, c.config.CallbackURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 只有在配置了 secret 时才添加签名头
	if c.config.Secret != "" {
		signature := generateSignature(payloadBytes, []byte(c.config.Secret))
		req.Header.Set("X-Webhook-Signature", signature)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ValidateWebhook validates an incoming webhook request
func (c *Client) ValidateWebhook(r *http.Request) (WebhookTriggerRequest, error) {
	var payload WebhookTriggerRequest

	// 如果配置了 secret，则验证签名
	if c.config.Secret != "" {
		signature := r.Header.Get("X-Webhook-Signature")
		if signature == "" {
			return payload, fmt.Errorf("missing webhook signature")
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			return payload, fmt.Errorf("failed to read request body: %w", err)
		}

		// Replace the request body for further processing
		r.Body = io.NopCloser(bytes.NewBuffer(body))

		expectedSignature := generateSignature(body, []byte(c.config.Secret))
		if signature != expectedSignature {
			return payload, fmt.Errorf("invalid webhook signature")
		}

		if err := json.Unmarshal(body, &payload); err != nil {
			return payload, fmt.Errorf("failed to unmarshal payload: %w", err)
		}
	} else {
		// 如果没有配置 secret，直接解码 JSON
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&payload); err != nil {
			return payload, fmt.Errorf("failed to decode payload: %w", err)
		}
	}

	// If reference is provided but branch is not, extract branch from reference
	if payload.Branch == "" && payload.Reference != "" {
		// Extract branch name from a git reference like "refs/heads/main"
		parts := strings.Split(payload.Reference, "/")
		if len(parts) >= 3 && parts[0] == "refs" && parts[1] == "heads" {
			payload.Branch = parts[2]
		}
	}

	return payload, nil
}

// generateSignature generates an HMAC signature for a webhook payload
func generateSignature(payload, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
