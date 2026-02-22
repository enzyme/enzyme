package linkpreview

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

// Repository handles link preview persistence.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new Repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// GetCachedURL returns the cache entry for a URL, or nil if not found / expired.
func (r *Repository) GetCachedURL(ctx context.Context, url string) (*CacheEntry, error) {
	var c CacheEntry
	var fetchedAt, expiresAt string
	var title, description, imageURL, siteName, fetchError sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT url, title, description, image_url, site_name, fetched_at, expires_at, fetch_error
		FROM link_preview_cache WHERE url = ?
	`, url).Scan(&c.URL, &title, &description, &imageURL, &siteName, &fetchedAt, &expiresAt, &fetchError)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	c.Title = title.String
	c.Description = description.String
	c.ImageURL = imageURL.String
	c.SiteName = siteName.String
	c.FetchError = fetchError.String
	c.FetchedAt, _ = time.Parse(time.RFC3339, fetchedAt)
	c.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)

	// Treat expired entries as a miss.
	if time.Now().After(c.ExpiresAt) {
		return nil, nil
	}

	return &c, nil
}

// SetCachedURL inserts or replaces a cache entry.
func (r *Repository) SetCachedURL(ctx context.Context, c *CacheEntry) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO link_preview_cache (url, title, description, image_url, site_name, fetched_at, expires_at, fetch_error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, c.URL, nullString(c.Title), nullString(c.Description), nullString(c.ImageURL), nullString(c.SiteName),
		c.FetchedAt.Format(time.RFC3339), c.ExpiresAt.Format(time.RFC3339), nullString(c.FetchError))
	return err
}

// CreatePreview inserts a per-message preview row. Duplicate message_id is silently ignored.
func (r *Repository) CreatePreview(ctx context.Context, p *Preview) error {
	if p.ID == "" {
		p.ID = ulid.Make().String()
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO link_previews (id, message_id, url, title, description, image_url, site_name, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.MessageID, p.URL, nullString(p.Title), nullString(p.Description), nullString(p.ImageURL), nullString(p.SiteName),
		p.CreatedAt.Format(time.RFC3339))
	return err
}

// GetForMessage returns the preview for a single message, or nil.
func (r *Repository) GetForMessage(ctx context.Context, messageID string) (*Preview, error) {
	var p Preview
	var title, description, imageURL, siteName sql.NullString
	var createdAt string

	err := r.db.QueryRowContext(ctx, `
		SELECT id, message_id, url, title, description, image_url, site_name, created_at
		FROM link_previews WHERE message_id = ?
	`, messageID).Scan(&p.ID, &p.MessageID, &p.URL, &title, &description, &imageURL, &siteName, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	p.Title = title.String
	p.Description = description.String
	p.ImageURL = imageURL.String
	p.SiteName = siteName.String
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &p, nil
}

// ListForMessages returns previews for multiple messages, keyed by message ID.
func (r *Repository) ListForMessages(ctx context.Context, messageIDs []string) (map[string]*Preview, error) {
	if len(messageIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(messageIDs))
	args := make([]interface{}, len(messageIDs))
	for i, id := range messageIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := `
		SELECT id, message_id, url, title, description, image_url, site_name, created_at
		FROM link_previews
		WHERE message_id IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*Preview)
	for rows.Next() {
		var p Preview
		var title, description, imageURL, siteName sql.NullString
		var createdAt string

		if err := rows.Scan(&p.ID, &p.MessageID, &p.URL, &title, &description, &imageURL, &siteName, &createdAt); err != nil {
			return nil, err
		}

		p.Title = title.String
		p.Description = description.String
		p.ImageURL = imageURL.String
		p.SiteName = siteName.String
		p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

		result[p.MessageID] = &p
	}

	return result, rows.Err()
}

// DeleteForMessage removes the preview for a message.
func (r *Repository) DeleteForMessage(ctx context.Context, messageID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM link_previews WHERE message_id = ?`, messageID)
	return err
}

// CleanExpiredCache removes expired entries from the cache table.
func (r *Repository) CleanExpiredCache(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM link_preview_cache WHERE expires_at < ?`, time.Now().UTC().Format(time.RFC3339))
	return err
}

// nullString returns sql.NullString for optional text fields.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
