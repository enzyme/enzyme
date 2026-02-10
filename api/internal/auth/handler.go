package auth

import "net/http"

// WorkspaceRepoForAuth is the interface needed by the handler to fetch workspace summaries.
type WorkspaceRepoForAuth interface {
	GetWorkspacesForUser(r *http.Request, userID string) ([]WorkspaceSummary, error)
}

// WorkspaceSummary represents a workspace a user belongs to.
type WorkspaceSummary struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	IconURL   *string `json:"icon_url,omitempty"`
	Role      string  `json:"role"`
	SortOrder *int    `json:"sort_order,omitempty"`
}
