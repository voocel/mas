//go:build sqlite

package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore implements StateStore interface using SQLite database
type SQLiteStore struct {
	db       *sql.DB
	dbPath   string
	mu       sync.RWMutex
	prepared map[string]*sql.Stmt
}

// SQLiteConfig contains configuration for SQLite store
type SQLiteConfig struct {
	// Database file path
	DBPath string `json:"db_path"`

	// WAL mode for better concurrency
	WALMode bool `json:"wal_mode"`

	// Connection pool settings
	MaxOpenConns    int           `json:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`

	// Performance settings
	CacheSize    int           `json:"cache_size"`    // SQLite cache size in KB
	SyncMode     int           `json:"sync_mode"`     // Synchronous mode (0=OFF, 1=NORMAL, 2=FULL)
	BusyTimeout  int           `json:"busy_timeout"`  // Busy timeout in milliseconds
	ForeignKeys  bool          `json:"foreign_keys"`  // Enable foreign key constraints
	QueryTimeout time.Duration `json:"query_timeout"` // Query timeout

	// Maintenance settings
	AutoVacuum      bool          `json:"auto_vacuum"`      // Enable auto vacuum
	CleanupInterval time.Duration `json:"cleanup_interval"` // Cleanup expired records interval
}

// DefaultSQLiteConfig returns a default SQLite configuration
func DefaultSQLiteConfig() SQLiteConfig {
	return SQLiteConfig{
		DBPath:          "./checkpoints.db",
		WALMode:         true,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 15 * time.Minute,
		CacheSize:       10000, // 10MB cache
		SyncMode:        1,     // NORMAL
		BusyTimeout:     5000,  // 5 seconds
		ForeignKeys:     true,
		QueryTimeout:    30 * time.Second,
		AutoVacuum:      true,
		CleanupInterval: time.Hour,
	}
}

// NewSQLiteStore creates a new SQLite-based state store
func NewSQLiteStore(config SQLiteConfig) (*SQLiteStore, error) {
	if err := ensureDir(filepath.Dir(config.DBPath)); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	connStr := config.DBPath + "?"
	if config.WALMode {
		connStr += "journal_mode=WAL&"
	}
	connStr += fmt.Sprintf("cache_size=-%d&", config.CacheSize) // Negative for KB
	connStr += fmt.Sprintf("synchronous=%d&", config.SyncMode)
	connStr += fmt.Sprintf("busy_timeout=%d&", config.BusyTimeout)
	if config.ForeignKeys {
		connStr += "foreign_keys=ON&"
	}
	if config.AutoVacuum {
		connStr += "auto_vacuum=INCREMENTAL&"
	}

	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping SQLite database: %w", err)
	}

	store := &SQLiteStore{
		db:       db,
		dbPath:   config.DBPath,
		prepared: make(map[string]*sql.Stmt),
	}

	// Initialize schema
	if err := store.initSchema(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	if err := store.prepareStatements(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to prepare statements: %w", err)
	}

	if config.CleanupInterval > 0 {
		go store.cleanupRoutine(config.CleanupInterval)
	}

	return store, nil
}

// initSchema creates the necessary tables and indexes
func (ss *SQLiteStore) initSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS checkpoints (
		key TEXT PRIMARY KEY,
		data BLOB NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		expires_at INTEGER
	);

	CREATE INDEX IF NOT EXISTS idx_expires ON checkpoints(expires_at);
	CREATE INDEX IF NOT EXISTS idx_created ON checkpoints(created_at);

	-- Table for metadata and versioning
	CREATE TABLE IF NOT EXISTS store_metadata (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at INTEGER NOT NULL
	);

	INSERT OR IGNORE INTO store_metadata (key, value, updated_at) 
	VALUES ('schema_version', '1', ?);
	`

	_, err := ss.db.ExecContext(ctx, schema, time.Now().Unix())
	return err
}

// prepareStatements prepares commonly used SQL statements
func (ss *SQLiteStore) prepareStatements() error {
	statements := map[string]string{
		"put": `
			INSERT OR REPLACE INTO checkpoints (key, data, created_at, updated_at, expires_at)
			VALUES (?, ?, 
				COALESCE((SELECT created_at FROM checkpoints WHERE key = ?), ?),
				?, ?)`,
		"get":        "SELECT data FROM checkpoints WHERE key = ? AND (expires_at IS NULL OR expires_at > ?)",
		"delete":     "DELETE FROM checkpoints WHERE key = ?",
		"exists":     "SELECT 1 FROM checkpoints WHERE key = ? AND (expires_at IS NULL OR expires_at > ?) LIMIT 1",
		"list":       "SELECT key FROM checkpoints WHERE key LIKE ? AND (expires_at IS NULL OR expires_at > ?) ORDER BY created_at DESC",
		"cleanup":    "DELETE FROM checkpoints WHERE expires_at IS NOT NULL AND expires_at <= ?",
		"count":      "SELECT COUNT(*) FROM checkpoints WHERE expires_at IS NULL OR expires_at > ?",
		"total_size": "SELECT SUM(LENGTH(data)) FROM checkpoints WHERE expires_at IS NULL OR expires_at > ?",
	}

	for name, query := range statements {
		stmt, err := ss.db.Prepare(query)
		if err != nil {
			return fmt.Errorf("failed to prepare statement %s: %w", name, err)
		}
		ss.prepared[name] = stmt
	}

	return nil
}

// Put stores data with the given key
func (ss *SQLiteStore) Put(ctx context.Context, key string, data []byte) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	now := time.Now().Unix()

	_, err := ss.prepared["put"].ExecContext(ctx, key, data, key, now, now, nil)
	if err != nil {
		return fmt.Errorf("failed to put data: %w", err)
	}

	return nil
}

// Get retrieves data by key
func (ss *SQLiteStore) Get(ctx context.Context, key string) ([]byte, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	var data []byte
	err := ss.prepared["get"].QueryRowContext(ctx, key, time.Now().Unix()).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("key not found: %s", key)
		}
		return nil, fmt.Errorf("failed to get data: %w", err)
	}

	return data, nil
}

// Delete removes data by key
func (ss *SQLiteStore) Delete(ctx context.Context, key string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	_, err := ss.prepared["delete"].ExecContext(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete data: %w", err)
	}

	return nil
}

// List returns all keys with the given prefix
func (ss *SQLiteStore) List(ctx context.Context, prefix string) ([]string, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	pattern := prefix + "%"
	rows, err := ss.prepared["list"].QueryContext(ctx, pattern, time.Now().Unix())
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("failed to scan key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return keys, nil
}

// Exists checks if a key exists
func (ss *SQLiteStore) Exists(ctx context.Context, key string) (bool, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	var exists int
	err := ss.prepared["exists"].QueryRowContext(ctx, key, time.Now().Unix()).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return exists == 1, nil
}

// Close releases any resources held by the store
func (ss *SQLiteStore) Close() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	// Close prepared statements
	for _, stmt := range ss.prepared {
		stmt.Close()
	}

	// Close database
	return ss.db.Close()
}

// PutWithTTL stores data with a TTL
func (ss *SQLiteStore) PutWithTTL(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	now := time.Now().Unix()
	expiresAt := now + int64(ttl.Seconds())

	stmt := `
		INSERT OR REPLACE INTO checkpoints (key, data, created_at, updated_at, expires_at)
		VALUES (?, ?, 
			COALESCE((SELECT created_at FROM checkpoints WHERE key = ?), ?),
			?, ?)`

	_, err := ss.db.ExecContext(ctx, stmt, key, data, key, now, now, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to put data with TTL: %w", err)
	}

	return nil
}

// BatchPut stores multiple key-value pairs in a transaction
func (ss *SQLiteStore) BatchPut(ctx context.Context, items map[string][]byte) error {
	if len(items) == 0 {
		return nil
	}

	ss.mu.Lock()
	defer ss.mu.Unlock()

	tx, err := ss.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO checkpoints (key, data, created_at, updated_at, expires_at)
		VALUES (?, ?, 
			COALESCE((SELECT created_at FROM checkpoints WHERE key = ?), ?),
			?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare batch statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now().Unix()
	for key, data := range items {
		_, err := stmt.ExecContext(ctx, key, data, key, now, now, nil)
		if err != nil {
			return fmt.Errorf("failed to execute batch statement for key %s: %w", key, err)
		}
	}

	return tx.Commit()
}

// BatchDelete removes multiple keys in a transaction
func (ss *SQLiteStore) BatchDelete(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	ss.mu.Lock()
	defer ss.mu.Unlock()

	tx, err := ss.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Build IN clause
	placeholders := strings.Repeat("?,", len(keys))
	placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma

	query := fmt.Sprintf("DELETE FROM checkpoints WHERE key IN (%s)", placeholders)

	args := make([]interface{}, len(keys))
	for i, key := range keys {
		args[i] = key
	}

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to batch delete: %w", err)
	}

	return tx.Commit()
}

// Cleanup removes expired records
func (ss *SQLiteStore) Cleanup(ctx context.Context) (int64, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	result, err := ss.prepared["cleanup"].ExecContext(ctx, time.Now().Unix())
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired records: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return affected, nil
}

// Stats returns store statistics
func (ss *SQLiteStore) Stats(ctx context.Context) (map[string]interface{}, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	var count int64
	var totalSize int64
	now := time.Now().Unix()

	err := ss.prepared["count"].QueryRowContext(ctx, now).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to get count: %w", err)
	}

	err = ss.prepared["total_size"].QueryRowContext(ctx, now).Scan(&totalSize)
	if err != nil {
		totalSize = 0 // If NULL, default to 0
	}

	stats := map[string]interface{}{
		"type":          "sqlite",
		"db_path":       ss.dbPath,
		"count":         count,
		"total_size":    totalSize,
		"total_size_mb": float64(totalSize) / 1024 / 1024,
	}

	return stats, nil
}

// cleanupRoutine runs periodic cleanup of expired records
func (ss *SQLiteStore) cleanupRoutine(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			affected, err := ss.Cleanup(ctx)
			cancel()

			if err != nil {
				fmt.Printf("SQLite cleanup error: %v\n", err)
			} else if affected > 0 {
				fmt.Printf("SQLite cleanup: removed %d expired records\n", affected)
			}
		}
	}
}

// Vacuum performs database maintenance
func (ss *SQLiteStore) Vacuum(ctx context.Context) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	_, err := ss.db.ExecContext(ctx, "VACUUM;")
	if err != nil {
		return fmt.Errorf("failed to vacuum database: %w", err)
	}

	return nil
}

// ensureDir creates directory if it doesn't exist
func ensureDir(dir string) error {
	if dir == "" || dir == "." {
		return nil
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	return nil
}
