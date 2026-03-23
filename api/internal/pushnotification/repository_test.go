package pushnotification

import (
	"context"
	"testing"
	"time"

	"github.com/enzyme/api/internal/testutil"
)

func TestUpsert(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ctx := context.Background()

	t.Run("insert new token", func(t *testing.T) {
		token := &DeviceToken{
			UserID:   user.ID,
			Token:    "fcm-token-1",
			Platform: "fcm",
			DeviceID: "device-1",
		}
		err := repo.Upsert(ctx, token)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token.ID == "" {
			t.Fatal("expected ID to be set")
		}
	})

	t.Run("upsert same token updates fields", func(t *testing.T) {
		token := &DeviceToken{
			UserID:   user.ID,
			Token:    "fcm-token-1",
			Platform: "fcm",
			DeviceID: "device-2", // different device
		}
		err := repo.Upsert(ctx, token)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify the device_id was updated
		tokens, err := repo.ListByUserID(ctx, user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		found := false
		for _, tk := range tokens {
			if tk.Token == "fcm-token-1" {
				found = true
				if tk.DeviceID != "device-2" {
					t.Errorf("expected device_id to be updated to device-2, got %s", tk.DeviceID)
				}
			}
		}
		if !found {
			t.Fatal("expected to find token fcm-token-1")
		}
	})

	t.Run("different user same token value", func(t *testing.T) {
		user2 := testutil.CreateTestUser(t, db, "other@example.com", "Other")
		token := &DeviceToken{
			UserID:   user2.ID,
			Token:    "fcm-token-1", // same token value, different user
			Platform: "fcm",
			DeviceID: "device-1",
		}
		err := repo.Upsert(ctx, token)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestDelete(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ctx := context.Background()

	// Insert a token
	token := &DeviceToken{
		UserID:   user.ID,
		Token:    "token-to-delete",
		Platform: "apns",
		DeviceID: "device-1",
	}
	if err := repo.Upsert(ctx, token); err != nil {
		t.Fatalf("setup: %v", err)
	}

	t.Run("delete existing token", func(t *testing.T) {
		err := repo.Delete(ctx, user.ID, "token-to-delete")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("delete non-existent token returns error", func(t *testing.T) {
		err := repo.Delete(ctx, user.ID, "no-such-token")
		if err != ErrTokenNotFound {
			t.Fatalf("expected ErrTokenNotFound, got %v", err)
		}
	})
}

func TestDeleteByDeviceID(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ctx := context.Background()

	// Insert two tokens for same device
	for _, tok := range []string{"token-a", "token-b"} {
		if err := repo.Upsert(ctx, &DeviceToken{
			UserID: user.ID, Token: tok, Platform: "fcm", DeviceID: "device-x",
		}); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}
	// Insert one token for a different device
	if err := repo.Upsert(ctx, &DeviceToken{
		UserID: user.ID, Token: "token-c", Platform: "fcm", DeviceID: "device-y",
	}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := repo.DeleteByDeviceID(ctx, user.ID, "device-x"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tokens, _ := repo.ListByUserID(ctx, user.ID)
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token remaining, got %d", len(tokens))
	}
	if tokens[0].Token != "token-c" {
		t.Errorf("expected token-c to remain, got %s", tokens[0].Token)
	}
}

func TestListByUserID(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ctx := context.Background()

	t.Run("empty list", func(t *testing.T) {
		tokens, err := repo.ListByUserID(ctx, user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tokens) != 0 {
			t.Fatalf("expected 0 tokens, got %d", len(tokens))
		}
	})

	// Add tokens
	for _, tok := range []string{"t1", "t2", "t3"} {
		if err := repo.Upsert(ctx, &DeviceToken{
			UserID: user.ID, Token: tok, Platform: "fcm", DeviceID: "d1",
		}); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	t.Run("returns all tokens", func(t *testing.T) {
		tokens, err := repo.ListByUserID(ctx, user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tokens) != 3 {
			t.Fatalf("expected 3 tokens, got %d", len(tokens))
		}
	})
}

func TestHasTokens(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ctx := context.Background()

	t.Run("no tokens", func(t *testing.T) {
		has, err := repo.HasTokens(ctx, user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if has {
			t.Fatal("expected false, got true")
		}
	})

	// Add a token
	if err := repo.Upsert(ctx, &DeviceToken{
		UserID: user.ID, Token: "t1", Platform: "fcm", DeviceID: "d1",
	}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	t.Run("has tokens", func(t *testing.T) {
		has, err := repo.HasTokens(ctx, user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !has {
			t.Fatal("expected true, got false")
		}
	})
}

func TestCleanupStale(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ctx := context.Background()

	// Insert a token
	if err := repo.Upsert(ctx, &DeviceToken{
		UserID: user.ID, Token: "old-token", Platform: "fcm", DeviceID: "d1",
	}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Manually backdate the token's updated_at
	_, err := db.ExecContext(ctx, `UPDATE device_tokens SET updated_at = ? WHERE token = ?`,
		time.Now().Add(-100*24*time.Hour).Format(time.RFC3339), "old-token")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Insert a fresh token
	if err := repo.Upsert(ctx, &DeviceToken{
		UserID: user.ID, Token: "new-token", Platform: "fcm", DeviceID: "d2",
	}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Cleanup tokens older than 90 days
	n, err := repo.CleanupStale(ctx, time.Now().Add(-90*24*time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 token cleaned up, got %d", n)
	}

	// Verify only fresh token remains
	tokens, _ := repo.ListByUserID(ctx, user.ID)
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token remaining, got %d", len(tokens))
	}
	if tokens[0].Token != "new-token" {
		t.Errorf("expected new-token to remain, got %s", tokens[0].Token)
	}
}

func TestDeleteToken(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ctx := context.Background()

	if err := repo.Upsert(ctx, &DeviceToken{
		UserID: user.ID, Token: "invalid-token", Platform: "apns", DeviceID: "d1",
	}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// DeleteToken removes by token value regardless of user
	if err := repo.DeleteToken(ctx, "invalid-token"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tokens, _ := repo.ListByUserID(ctx, user.ID)
	if len(tokens) != 0 {
		t.Fatalf("expected 0 tokens, got %d", len(tokens))
	}
}
