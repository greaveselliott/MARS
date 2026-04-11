package github

import (
	"encoding/json"
	"time"
)

// AuthMode controls how the client authenticates with the GitHub API.
type AuthMode string

const (
	AuthApp AuthMode = "app" // GitHub App JWT → installation token
	AuthPAT AuthMode = "pat" // Personal Access Token
)

// CheckStatus represents the status of a GitHub check run.
type CheckStatus string

const (
	CheckQueued     CheckStatus = "queued"
	CheckInProgress CheckStatus = "in_progress"
	CheckCompleted  CheckStatus = "completed"
)

// PullRequest represents a GitHub pull request (subset of fields we use).
type PullRequest struct {
	ID        int64  `json:"id"`
	Number    int    `json:"number"`
	HTMLURL   string `json:"html_url"`
	State     string `json:"state"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Head      PRRef  `json:"head"`
	Base      PRRef  `json:"base"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// PRRef is a branch reference within a PullRequest.
type PRRef struct {
	Ref  string     `json:"ref"`
	SHA  string     `json:"sha"`
	Repo *RepoShort `json:"repo,omitempty"`
}

// RepoShort is a minimal repository representation.
type RepoShort struct {
	FullName string `json:"full_name"`
}

// CheckRun represents a GitHub check run.
type CheckRun struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion,omitempty"`
	HTMLURL    string `json:"html_url"`
}

// installationToken is the response from the installation access token endpoint.
type installationToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// AppCredentials holds the result of a GitHub App manifest flow.
type AppCredentials struct {
	AppID         int64  `json:"app_id"`
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	PEM           string `json:"pem"`
	WebhookSecret string `json:"webhook_secret"`
}

// Event is a normalized webhook event.
type Event struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Action   string          `json:"action"`
	Repo     string          `json:"repo"`
	Payload  json.RawMessage `json:"payload"`
	Received time.Time       `json:"received"`
}
