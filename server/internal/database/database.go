package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

// Options controls SQLite connection pool and pragma settings.
type Options struct {
	MaxOpenConns int   // max open connections (default: 10)
	BusyTimeout  int   // milliseconds to wait on lock (default: 5000)
	CacheSize    int   // negative = KB, positive = pages (default: -2000)
	MmapSize     int64 // bytes, 0 = disabled (default: 0)
}

func Open(path string, opts Options) (*DB, error) {
	// Ensure the directory exists (skip for in-memory databases)
	if path != ":memory:" {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("creating database directory: %w", err)
		}
	}

	// Build DSN with pragmas applied per-connection, ensuring every connection
	// in the pool gets all pragmas (fixes foreign_keys correctness with pool > 1).
	// _txlock=immediate ensures BEGIN acquires a write lock immediately rather
	// than deferring it. Without this, the busy_timeout handler is never
	// invoked when upgrading from a read to write lock mid-transaction —
	// SQLite returns SQLITE_BUSY instantly to avoid deadlocks.
	// temp_store=2 uses memory-backed temp tables instead of disk, avoiding
	// temp file I/O for ORDER BY, GROUP BY, and window functions.
	dsn := fmt.Sprintf("%s?_txlock=immediate&_pragma=journal_mode%%28WAL%%29&_pragma=busy_timeout%%28%d%%29&_pragma=foreign_keys%%28ON%%29&_pragma=synchronous%%28NORMAL%%29&_pragma=cache_size%%28%d%%29&_pragma=mmap_size%%28%d%%29&_pragma=temp_store%%282%%29",
		path, opts.BusyTimeout, opts.CacheSize, opts.MmapSize)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db.SetMaxOpenConns(opts.MaxOpenConns)

	// Verify WAL mode took effect. journal_mode returns a result rather than
	// silently applying, so DSN-based setting can't be confirmed without a
	// read-back check. Skip for :memory: databases which don't support WAL.
	if path != ":memory:" {
		var journalMode string
		if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("checking journal_mode: %w", err)
		}
		if journalMode != "wal" {
			_ = db.Close()
			return nil, fmt.Errorf("expected journal_mode=wal, got %q", journalMode)
		}
	}

	return &DB{db}, nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

func (db *DB) Ping(ctx context.Context) error {
	return db.PingContext(ctx)
}

// Transaction helper
func (db *DB) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rolling back: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}
