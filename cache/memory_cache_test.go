package cache

import (
	"fmt"
	"testing"
)

func complexCache(cache *MemCache, max int) {
	for i := 0; i < max; i = i + 100000 {
		putBytes := make([]byte, 100000)
		cache.Put(i, i+100000, putBytes, "source-hash")
		fmt.Printf("size %v\n", cache.currentSize)
		fmt.Printf("elements %v\n", len(cache.sourceMap["source-hash"].list))
	}
}

func TestSearchExactMatch(t *testing.T) {
	cache := NewMemCache(1)

	putBytes := make([]byte, 30)
	cache.Put(50, 80, putBytes, "source-hash")

	fetchedBytes, err := cache.Get(50, 80, "source-hash")

	if (err != nil) && (len(fetchedBytes) != 30) {
		t.Errorf("Could not fetch the correct data")
	}
}

func TestSearchNOMatch(t *testing.T) {
	cache := NewMemCache(1)

	putBytes := make([]byte, 30)
	cache.Put(50, 80, putBytes, "source-hash")

	fetchedBytes, err := cache.Get(30, 40, "source-hash")

	if (err == ErrCacheMiss) && (fetchedBytes != nil) {
		t.Errorf("got a cache miss when data is not there")
	}
}

func TestSearchSubMatch(t *testing.T) {
	cache := NewMemCache(1)

	putBytes := make([]byte, 30)
	cache.Put(50, 80, putBytes, "source-hash")

	fetchedBytes, err := cache.Get(50, 70, "source-hash")

	if (err == nil) && (len(fetchedBytes) != 20) {
		t.Errorf("Could not fetch the correct data")
	}
}

func TestSearchExactMatchComplex(t *testing.T) {
	cache := NewMemCache(1)

	complexCache(cache, 100)

	fetchedBytes, err := cache.Get(70, 80, "source-hash")

	if (err != nil) && (len(fetchedBytes) != 10) {
		t.Errorf("Could not fetch the correct data")
	}
}

func TestEvictComplex(t *testing.T) {
	cache := NewMemCache(1)

	complexCache(cache, 1000000)

	putBytes := make([]byte, 50)
	cache.Put(400, 450, putBytes, "source-hash")

	fetchedBytes, err := cache.Get(0, 100000, "source-hash")

	if (err == ErrCacheMiss) && (fetchedBytes != nil) {
		t.Errorf("got a cache miss when data is not there")
	}
}
