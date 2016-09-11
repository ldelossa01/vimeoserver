package cache

import "errors"

var (
	// ErrCacheMiss cache miss
	ErrCacheMiss = errors.New("Value not in cache")
)

// Cache Interface for implementing a LRU cache
type Cache interface {
	// Put Place item into cache and handles evictions
	Put(start, end int, buffer []byte, sourceURL string) error
	// Get Retreives items from the cache
	Get(start, end int, sourceURL string) ([]byte, error)
}
