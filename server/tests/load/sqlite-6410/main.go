// Reproduction test for SQLITE_IOERR_ROLLBACK_ATOMIC (6410) errors.
//
// Creates a SQLite database with the messages table schema (matching production),
// optionally adds the unread scan index, and hammers it with concurrent reads
// and writes to see if 6410 errors appear.
//
// The read workload simulates the GetWorkspaceNotificationSummaries query that
// was being triggered by every SSE message.created event (thundering herd).
//
// Usage:
//
//	go run ./server/tests/load/sqlite-6410
//	go run ./server/tests/load/sqlite-6410 -index=partial
//	go run ./server/tests/load/sqlite-6410 -index=composite -writers=8 -readers=16 -messages=5000
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/oklog/ulid/v2"
	_ "modernc.org/sqlite"
)

var (
	indexType    = flag.String("index", "composite", "Index type to test: none, partial, composite")
	writers     = flag.Int("writers", 4, "Number of concurrent writer goroutines")
	readers     = flag.Int("readers", 8, "Number of concurrent reader goroutines (simulate notification queries)")
	messages    = flag.Int("messages", 2000, "Total messages to insert")
	maxConns    = flag.Int("max-conns", 4, "MaxOpenConns for the connection pool")
	busyTimeout = flag.Int("busy-timeout", 5000, "SQLite busy_timeout in ms")
	keepDB      = flag.Bool("keep-db", false, "Keep the database file after the test")
)

// Schema matches production messages table from migration 007 + later ALTERs.
const createSchema = `
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS workspaces (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS channels (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id),
    name TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'public',
    archived_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS channel_memberships (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    channel_id TEXT NOT NULL REFERENCES channels(id),
    channel_role TEXT,
    last_read_message_id TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS notification_preferences (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    channel_id TEXT NOT NULL REFERENCES channels(id),
    notify_level TEXT
);

CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    channel_id TEXT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    content TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'user',
    mentions TEXT NOT NULL DEFAULT '[]',
    thread_parent_id TEXT REFERENCES messages(id) ON DELETE CASCADE,
    reply_count INTEGER NOT NULL DEFAULT 0,
    last_reply_at TEXT,
    edited_at TEXT,
    deleted_at TEXT,
    pinned_at TEXT,
    pinned_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    system_event TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Existing production indexes
CREATE INDEX IF NOT EXISTS idx_messages_channel ON messages(channel_id, id);
CREATE INDEX IF NOT EXISTS idx_messages_thread ON messages(thread_parent_id, id);
CREATE INDEX IF NOT EXISTS idx_messages_type ON messages(channel_id, type);
CREATE INDEX IF NOT EXISTS idx_messages_pinned ON messages(channel_id, pinned_at) WHERE pinned_at IS NOT NULL;
`

// The exact query from GetWorkspaceNotificationSummaries in channel/repository.go
const notificationQuery = `
SELECT c.workspace_id,
       COALESCE(SUM(
           (SELECT COUNT(*) FROM messages m
            WHERE m.channel_id = c.id
              AND m.thread_parent_id IS NULL
              AND m.deleted_at IS NULL
              AND (cm.last_read_message_id IS NULL OR m.id > cm.last_read_message_id))
       ), 0) as unread_count,
       COALESCE(SUM(
           (SELECT COUNT(*) FROM messages m
            WHERE m.channel_id = c.id
              AND m.thread_parent_id IS NULL
              AND m.deleted_at IS NULL
              AND (cm.last_read_message_id IS NULL OR m.id > cm.last_read_message_id)
              AND CASE
                WHEN c.type IN ('dm', 'group_dm') THEN 1
                WHEN np.notify_level = 'none' THEN 0
                WHEN np.notify_level = 'all' THEN 1
                WHEN np.notify_level = 'mentions' OR np.notify_level IS NULL THEN
                  EXISTS (
                    SELECT 1 FROM json_each(m.mentions) je
                    WHERE je.value = ? OR je.value IN ('@channel', '@everyone')
                  )
                ELSE 0
              END = 1)
       ), 0) as notification_count
FROM channels c
JOIN channel_memberships cm ON cm.channel_id = c.id AND cm.user_id = ?
LEFT JOIN notification_preferences np ON np.channel_id = c.id AND np.user_id = ?
WHERE c.archived_at IS NULL
GROUP BY c.workspace_id
`

func main() {
	flag.Parse()

	// Create temp directory for the DB
	tmpDir, err := os.MkdirTemp("", "sqlite-6410-*")
	if err != nil {
		log.Fatalf("creating temp dir: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")

	if !*keepDB {
		defer os.RemoveAll(tmpDir)
	} else {
		fmt.Printf("Database path: %s\n", dbPath)
	}

	// Open with same DSN pattern as production
	dsn := fmt.Sprintf("%s?_pragma=journal_mode%%28WAL%%29&_pragma=busy_timeout%%28%d%%29&_pragma=foreign_keys%%28ON%%29&_pragma=synchronous%%28NORMAL%%29&_pragma=cache_size%%28-2000%%29",
		dbPath, *busyTimeout)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatalf("opening database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(*maxConns)

	// Verify WAL mode
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		log.Fatalf("checking journal_mode: %v", err)
	}
	if journalMode != "wal" {
		log.Fatalf("expected WAL mode, got %q", journalMode)
	}

	// Create schema
	if _, err := db.Exec(createSchema); err != nil {
		log.Fatalf("creating schema: %v", err)
	}

	// Seed test data
	userIDs := seedTestData(db)

	// Add the index under test
	switch *indexType {
	case "none":
		fmt.Println("Index: none (baseline)")
	case "partial":
		fmt.Println("Index: partial (WHERE thread_parent_id IS NULL AND deleted_at IS NULL)")
		_, err := db.Exec(`CREATE INDEX idx_messages_unread_scan ON messages(channel_id, id) WHERE thread_parent_id IS NULL AND deleted_at IS NULL`)
		if err != nil {
			log.Fatalf("creating partial index: %v", err)
		}
	case "composite":
		fmt.Println("Index: composite (channel_id, thread_parent_id, deleted_at, id)")
		_, err := db.Exec(`CREATE INDEX idx_messages_unread_scan ON messages(channel_id, thread_parent_id, deleted_at, id)`)
		if err != nil {
			log.Fatalf("creating composite index: %v", err)
		}
	default:
		log.Fatalf("unknown index type: %s (use none, partial, or composite)", *indexType)
	}

	fmt.Printf("Config: %d writers, %d readers, %d messages, max_conns=%d, busy_timeout=%dms\n",
		*writers, *readers, *messages, *maxConns, *busyTimeout)

	// Shared counters
	var (
		writeErrors  atomic.Int64
		write6410    atomic.Int64
		writeOther   atomic.Int64
		totalWritten atomic.Int64
		readErrors   atomic.Int64
		read6410     atomic.Int64
		readOther    atomic.Int64
		totalReads   atomic.Int64
		errorDetails sync.Map
		wg           sync.WaitGroup
	)

	ctx, cancel := context.WithCancel(context.Background())
	perWriter := *messages / *writers

	start := time.Now()

	// Start reader goroutines — they run the expensive notification query in a tight loop
	for r := 0; r < *readers; r++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			userID := userIDs[readerID%len(userIDs)]
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				rows, err := db.QueryContext(ctx, notificationQuery, userID, userID, userID)
				if err != nil {
					if ctx.Err() != nil {
						return // shutting down
					}
					readErrors.Add(1)
					errStr := err.Error()
					if strings.Contains(errStr, "6410") || strings.Contains(errStr, "ROLLBACK_ATOMIC") {
						read6410.Add(1)
					} else {
						readOther.Add(1)
					}
					errorDetails.LoadOrStore("read: "+errStr, totalWritten.Load())
					continue
				}
				for rows.Next() {
					var wsID string
					var unread, notif int
					rows.Scan(&wsID, &unread, &notif)
				}
				rows.Close()
				totalReads.Add(1)
			}
		}(r)
	}

	// Start writer goroutines
	for w := 0; w < *writers; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				msgID := ulid.Make().String()
				userID := userIDs[writerID%len(userIDs)]
				content := fmt.Sprintf("Message %d from writer %d", i, writerID)

				_, err := db.Exec(
					`INSERT INTO messages (id, channel_id, user_id, content, type, created_at, updated_at)
					 VALUES (?, ?, ?, ?, 'user', datetime('now'), datetime('now'))`,
					msgID, "ch-general", userID, content,
				)
				if err != nil {
					writeErrors.Add(1)
					errStr := err.Error()
					if strings.Contains(errStr, "6410") || strings.Contains(errStr, "ROLLBACK_ATOMIC") {
						write6410.Add(1)
					} else {
						writeOther.Add(1)
					}
					errorDetails.LoadOrStore("write: "+errStr, totalWritten.Load())
					continue
				}
				totalWritten.Add(1)

				if n := totalWritten.Load(); n%250 == 0 {
					fmt.Printf("  ... %d messages written, %d reads completed\n", n, totalReads.Load())
				}
			}
		}(w)
	}

	// Wait for writers to finish, then stop readers
	// (We need a separate mechanism since wg includes readers)
	writersDone := make(chan struct{})
	go func() {
		// Poll until all messages written or all writers errored out
		for {
			if totalWritten.Load()+writeErrors.Load() >= int64(*messages) {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		close(writersDone)
	}()

	<-writersDone
	cancel() // stop readers
	wg.Wait()
	elapsed := time.Since(start)

	// Results
	fmt.Printf("\n--- Results ---\n")
	fmt.Printf("Duration:       %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("Messages:       %d / %d written\n", totalWritten.Load(), *messages)
	fmt.Printf("Write rate:     %.0f msg/sec\n", float64(totalWritten.Load())/elapsed.Seconds())
	fmt.Printf("Reads:          %d completed\n", totalReads.Load())
	fmt.Printf("Read rate:      %.0f reads/sec\n", float64(totalReads.Load())/elapsed.Seconds())
	fmt.Printf("Write errors:   %d (6410: %d, other: %d)\n", writeErrors.Load(), write6410.Load(), writeOther.Load())
	fmt.Printf("Read errors:    %d (6410: %d, other: %d)\n", readErrors.Load(), read6410.Load(), readOther.Load())

	if writeErrors.Load()+readErrors.Load() > 0 {
		fmt.Printf("\nError details:\n")
		errorDetails.Range(func(key, value any) bool {
			fmt.Printf("  [after msg %d] %s\n", value.(int64), key.(string))
			return true
		})
	}

	total6410 := write6410.Load() + read6410.Load()
	if total6410 > 0 {
		fmt.Printf("\n*** 6410 REPRODUCED (%d occurrences) ***\n", total6410)
		os.Exit(1)
	} else {
		fmt.Printf("\nNo 6410 errors observed.\n")
	}
}

func seedTestData(db *sql.DB) []string {
	// Create a workspace
	db.Exec(`INSERT INTO workspaces (id, name) VALUES ('ws-1', 'Test Workspace')`)

	// Create multiple channels (more realistic — notification query joins across all channels)
	channels := []string{"ch-general", "ch-random", "ch-engineering", "ch-design", "ch-product"}
	for _, ch := range channels {
		db.Exec(`INSERT INTO channels (id, workspace_id, name, type) VALUES (?, 'ws-1', ?, 'public')`,
			ch, ch[3:]) // strip "ch-" prefix for name
	}

	// Create users
	userIDs := []string{"user-1", "user-2", "user-3", "user-4", "user-5", "user-6", "user-7", "user-8"}
	for _, id := range userIDs {
		db.Exec(`INSERT INTO users (id, email, display_name) VALUES (?, ?, ?)`,
			id, id+"@test.com", "User "+id)
	}

	// Create channel memberships (every user in every channel, like production)
	for _, userID := range userIDs {
		for _, ch := range channels {
			membershipID := ulid.Make().String()
			db.Exec(`INSERT INTO channel_memberships (id, user_id, channel_id) VALUES (?, ?, ?)`,
				membershipID, userID, ch)
		}
	}

	// Pre-seed messages across channels so notification query has work to do
	for i := 0; i < 500; i++ {
		msgID := ulid.Make().String()
		userID := userIDs[i%len(userIDs)]
		ch := channels[i%len(channels)]
		db.Exec(`INSERT INTO messages (id, channel_id, user_id, content, type, mentions) VALUES (?, ?, ?, ?, 'user', '[]')`,
			msgID, ch, userID, fmt.Sprintf("Seed message %d", i))
	}

	fmt.Printf("Seeded: 1 workspace, %d channels, %d users, %d memberships, 500 messages\n",
		len(channels), len(userIDs), len(channels)*len(userIDs))
	return userIDs
}
