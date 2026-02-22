package linkpreview

import (
	"context"
	"testing"
	"time"

	"github.com/enzyme/api/internal/testutil"
)

func TestGetCachedURL_Miss(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)

	entry, err := repo.GetCachedURL(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry != nil {
		t.Fatal("expected nil for cache miss")
	}
}

func TestSetAndGetCachedURL(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	entry := &CacheEntry{
		URL:         "https://example.com",
		Title:       "Example",
		Description: "An example site",
		ImageURL:    "https://example.com/image.png",
		SiteName:    "Example Site",
		FetchedAt:   now,
		ExpiresAt:   now.Add(CacheTTL),
	}
	if err := repo.SetCachedURL(ctx, entry); err != nil {
		t.Fatalf("SetCachedURL: %v", err)
	}

	got, err := repo.GetCachedURL(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("GetCachedURL: %v", err)
	}
	if got == nil {
		t.Fatal("expected cache entry, got nil")
	}
	if got.Title != "Example" {
		t.Errorf("title = %q, want %q", got.Title, "Example")
	}
	if got.Description != "An example site" {
		t.Errorf("description = %q, want %q", got.Description, "An example site")
	}
	if got.ImageURL != "https://example.com/image.png" {
		t.Errorf("image_url = %q, want %q", got.ImageURL, "https://example.com/image.png")
	}
	if got.SiteName != "Example Site" {
		t.Errorf("site_name = %q, want %q", got.SiteName, "Example Site")
	}
}

func TestGetCachedURL_Expired(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	past := time.Now().UTC().Add(-2 * time.Hour)
	entry := &CacheEntry{
		URL:       "https://expired.com",
		Title:     "Old",
		FetchedAt: past.Add(-CacheTTL),
		ExpiresAt: past, // already expired
	}
	if err := repo.SetCachedURL(ctx, entry); err != nil {
		t.Fatalf("SetCachedURL: %v", err)
	}

	got, err := repo.GetCachedURL(ctx, "https://expired.com")
	if err != nil {
		t.Fatalf("GetCachedURL: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for expired cache entry")
	}
}

func TestCreatePreview_Idempotent(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	// Create a test message to reference
	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", "public")
	msg := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "Check https://example.com")

	p1 := &Preview{
		MessageID: msg.ID,
		URL:       "https://example.com",
		Title:     "Example",
	}
	if err := repo.CreatePreview(ctx, p1); err != nil {
		t.Fatalf("first CreatePreview: %v", err)
	}

	// Second insert with same message_id should replace the first
	p2 := &Preview{
		MessageID: msg.ID,
		URL:       "https://other.com",
		Title:     "Other",
	}
	if err := repo.CreatePreview(ctx, p2); err != nil {
		t.Fatalf("second CreatePreview: %v", err)
	}

	// Should return the second (replaced) preview
	got, err := repo.GetForMessage(ctx, msg.ID)
	if err != nil {
		t.Fatalf("GetForMessage: %v", err)
	}
	if got == nil {
		t.Fatal("expected preview, got nil")
	}
	if got.URL != "https://other.com" {
		t.Errorf("url = %q, want %q", got.URL, "https://other.com")
	}
}

func TestListForMessages(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", "public")
	msg1 := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "https://a.com")
	msg2 := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "https://b.com")
	msg3 := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "no link here")

	_ = repo.CreatePreview(ctx, &Preview{MessageID: msg1.ID, URL: "https://a.com", Title: "A"})
	_ = repo.CreatePreview(ctx, &Preview{MessageID: msg2.ID, URL: "https://b.com", Title: "B"})

	result, err := repo.ListForMessages(ctx, []string{msg1.ID, msg2.ID, msg3.ID})
	if err != nil {
		t.Fatalf("ListForMessages: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 previews, got %d", len(result))
	}
	if result[msg1.ID].Title != "A" {
		t.Errorf("msg1 title = %q, want %q", result[msg1.ID].Title, "A")
	}
	if result[msg2.ID].Title != "B" {
		t.Errorf("msg2 title = %q, want %q", result[msg2.ID].Title, "B")
	}
	if result[msg3.ID] != nil {
		t.Error("msg3 should have no preview")
	}
}

func TestCleanExpiredCache(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	past := time.Now().UTC().Add(-2 * time.Hour)
	_ = repo.SetCachedURL(ctx, &CacheEntry{
		URL:       "https://expired.com",
		FetchedAt: past,
		ExpiresAt: past.Add(time.Hour), // still expired
	})
	_ = repo.SetCachedURL(ctx, &CacheEntry{
		URL:       "https://fresh.com",
		FetchedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(CacheTTL),
	})

	if err := repo.CleanExpiredCache(ctx); err != nil {
		t.Fatalf("CleanExpiredCache: %v", err)
	}

	// Expired should be gone (raw query since GetCachedURL also filters by time)
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM link_preview_cache").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 remaining cache entry, got %d", count)
	}
}
