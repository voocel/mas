//go:build redis

package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements StateStore interface using Redis
type RedisStore struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

// RedisConfig contains configuration for Redis store
type RedisConfig struct {
	// Redis connection options
	Addrs    []string `json:"addrs"`    // Redis server addresses
	Username string   `json:"username"` // Username for ACL
	Password string   `json:"password"` // Password
	DB       int      `json:"db"`       // Database number

	// Cluster options
	ClusterMode bool `json:"cluster_mode"` // Enable cluster mode

	// Connection pool options
	PoolSize        int           `json:"pool_size"`          // Connection pool size
	MinIdleConns    int           `json:"min_idle_conns"`     // Minimum idle connections
	MaxIdleConns    int           `json:"max_idle_conns"`     // Maximum idle connections
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`  // Connection max lifetime
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"` // Connection max idle time

	// Timeouts
	DialTimeout  time.Duration `json:"dial_timeout"`  // Dial timeout
	ReadTimeout  time.Duration `json:"read_timeout"`  // Read timeout
	WriteTimeout time.Duration `json:"write_timeout"` // Write timeout

	// Key options
	KeyPrefix string        `json:"key_prefix"` // Key prefix for namespacing
	TTL       time.Duration `json:"ttl"`        // Default TTL for keys

	// Retry options
	MaxRetries      int           `json:"max_retries"`       // Maximum retry attempts
	MinRetryBackoff time.Duration `json:"min_retry_backoff"` // Minimum retry backoff
	MaxRetryBackoff time.Duration `json:"max_retry_backoff"` // Maximum retry backoff
}

// DefaultRedisConfig returns a default Redis configuration
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		Addrs:           []string{"localhost:6379"},
		DB:              0,
		ClusterMode:     false,
		PoolSize:        10,
		MinIdleConns:    2,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 30 * time.Minute,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		KeyPrefix:       "mas:checkpoint:",
		TTL:             24 * time.Hour,
		MaxRetries:      3,
		MinRetryBackoff: 100 * time.Millisecond,
		MaxRetryBackoff: 2 * time.Second,
	}
}

// NewRedisStore creates a new Redis-based state store
func NewRedisStore(config RedisConfig) (*RedisStore, error) {
	var client redis.UniversalClient

	if config.ClusterMode {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:           config.Addrs,
			Username:        config.Username,
			Password:        config.Password,
			PoolSize:        config.PoolSize,
			MinIdleConns:    config.MinIdleConns,
			MaxIdleConns:    config.MaxIdleConns,
			ConnMaxLifetime: config.ConnMaxLifetime,
			ConnMaxIdleTime: config.ConnMaxIdleTime,
			DialTimeout:     config.DialTimeout,
			ReadTimeout:     config.ReadTimeout,
			WriteTimeout:    config.WriteTimeout,
			MaxRetries:      config.MaxRetries,
			MinRetryBackoff: config.MinRetryBackoff,
			MaxRetryBackoff: config.MaxRetryBackoff,
		})
	} else {
		// Single instance or sentinel
		if len(config.Addrs) == 1 {
			client = redis.NewClient(&redis.Options{
				Addr:            config.Addrs[0],
				Username:        config.Username,
				Password:        config.Password,
				DB:              config.DB,
				PoolSize:        config.PoolSize,
				MinIdleConns:    config.MinIdleConns,
				MaxIdleConns:    config.MaxIdleConns,
				ConnMaxLifetime: config.ConnMaxLifetime,
				ConnMaxIdleTime: config.ConnMaxIdleTime,
				DialTimeout:     config.DialTimeout,
				ReadTimeout:     config.ReadTimeout,
				WriteTimeout:    config.WriteTimeout,
				MaxRetries:      config.MaxRetries,
				MinRetryBackoff: config.MinRetryBackoff,
				MaxRetryBackoff: config.MaxRetryBackoff,
			})
		} else {
			// Sentinel mode
			client = redis.NewFailoverClient(&redis.FailoverOptions{
				MasterName:      "master", // Default master name
				SentinelAddrs:   config.Addrs,
				Username:        config.Username,
				Password:        config.Password,
				DB:              config.DB,
				PoolSize:        config.PoolSize,
				MinIdleConns:    config.MinIdleConns,
				MaxIdleConns:    config.MaxIdleConns,
				ConnMaxLifetime: config.ConnMaxLifetime,
				ConnMaxIdleTime: config.ConnMaxIdleTime,
				DialTimeout:     config.DialTimeout,
				ReadTimeout:     config.ReadTimeout,
				WriteTimeout:    config.WriteTimeout,
				MaxRetries:      config.MaxRetries,
				MinRetryBackoff: config.MinRetryBackoff,
				MaxRetryBackoff: config.MaxRetryBackoff,
			})
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStore{
		client: client,
		prefix: config.KeyPrefix,
		ttl:    config.TTL,
	}, nil
}

// Put stores data with the given key
func (rs *RedisStore) Put(ctx context.Context, key string, data []byte) error {
	fullKey := rs.prefix + key
	err := rs.client.Set(ctx, fullKey, data, rs.ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to put data to Redis: %w", err)
	}

	return nil
}

// Get retrieves data by key
func (rs *RedisStore) Get(ctx context.Context, key string) ([]byte, error) {
	fullKey := rs.prefix + key
	data, err := rs.client.Get(ctx, fullKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("key not found: %s", key)
		}
		return nil, fmt.Errorf("failed to get data from Redis: %w", err)
	}

	return data, nil
}

// Delete removes data by key
func (rs *RedisStore) Delete(ctx context.Context, key string) error {
	fullKey := rs.prefix + key
	err := rs.client.Del(ctx, fullKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete data from Redis: %w", err)
	}

	return nil
}

// List returns all keys with the given prefix
func (rs *RedisStore) List(ctx context.Context, prefix string) ([]string, error) {
	fullPrefix := rs.prefix + prefix

	var keys []string
	iter := rs.client.Scan(ctx, 0, fullPrefix+"*", 0).Iterator()

	for iter.Next(ctx) {
		fullKey := iter.Val()
		if strings.HasPrefix(fullKey, rs.prefix) {
			originalKey := strings.TrimPrefix(fullKey, rs.prefix)
			keys = append(keys, originalKey)
		}
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan keys in Redis: %w", err)
	}

	return keys, nil
}

// Exists checks if a key exists
func (rs *RedisStore) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := rs.prefix + key
	exists, err := rs.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence in Redis: %w", err)
	}

	return exists > 0, nil
}

// Close releases any resources held by the store
func (rs *RedisStore) Close() error {
	return rs.client.Close()
}

// BatchPut stores multiple key-value pairs in a single operation
func (rs *RedisStore) BatchPut(ctx context.Context, items map[string][]byte) error {
	if len(items) == 0 {
		return nil
	}

	pipe := rs.client.Pipeline()

	for key, data := range items {
		fullKey := rs.prefix + key
		pipe.Set(ctx, fullKey, data, rs.ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to batch put data to Redis: %w", err)
	}

	return nil
}

// BatchDelete removes multiple keys in a single operation
func (rs *RedisStore) BatchDelete(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = rs.prefix + key
	}

	err := rs.client.Del(ctx, fullKeys...).Err()
	if err != nil {
		return fmt.Errorf("failed to batch delete data from Redis: %w", err)
	}

	return nil
}

// SetTTL sets TTL for a specific key
func (rs *RedisStore) SetTTL(ctx context.Context, key string, ttl time.Duration) error {
	fullKey := rs.prefix + key

	err := rs.client.Expire(ctx, fullKey, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set TTL for key in Redis: %w", err)
	}

	return nil
}

// GetTTL gets remaining TTL for a key
func (rs *RedisStore) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := rs.prefix + key

	ttl, err := rs.client.TTL(ctx, fullKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL for key in Redis: %w", err)
	}

	return ttl, nil
}

// Stats returns Redis store statistics
func (rs *RedisStore) Stats(ctx context.Context) (map[string]interface{}, error) {
	info, err := rs.client.Info(ctx, "memory", "stats", "keyspace").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis stats: %w", err)
	}

	stats := map[string]interface{}{
		"type":   "redis",
		"prefix": rs.prefix,
		"ttl":    rs.ttl.String(),
		"info":   info,
	}

	return stats, nil
}
