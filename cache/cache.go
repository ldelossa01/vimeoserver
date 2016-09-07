package cache

import (
	"container/heap"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

var (
	// ErrCacheMiss cache miss
	ErrCacheMiss = errors.New("Value not in cache")
)

type metaList struct {
	list []*metaObject
}

type lruHeap []*lruObject

func (m *metaList) Len() int           { return len(m.list) }
func (m *metaList) Swap(i, j int)      { m.list[i], m.list[j] = m.list[j], m.list[i] }
func (m *metaList) Less(i, j int) bool { return m.list[i].start < m.list[j].start }
func (m *metaList) append(newEle *metaObject) {
	m.list = append(m.list, newEle)
	sort.Sort(m)
}

func (lh lruHeap) Len() int           { return len(lh) }
func (lh lruHeap) Less(i, j int) bool { return lh[i].epoch < lh[j].epoch }
func (lh lruHeap) Swap(i, j int)      { lh[i], lh[j] = lh[j], lh[i] }
func (lh *lruHeap) Push(x interface{}) {
	item := x.(*lruObject)
	*lh = append(*lh, item)
}
func (lh *lruHeap) Pop() interface{} {
	old := *lh
	n := len(old)
	item := old[n-1]
	*lh = old[0 : n-1]
	return item
}

// Cache is a cache
type Cache struct {
	maxSize     int
	currentSize int
	meta        *metaList
	lru         *lruHeap
	lock        sync.Mutex
}

type metaObject struct {
	start  int
	end    int
	lru    *lruObject
	size   int
	buffer []byte
}

type lruObject struct {
	epoch int64
	ptr   *metaObject
}

// NewCache cache factory
func NewCache(sizeMb int) *Cache {
	size := sizeMb * 1000000
	lh := &lruHeap{}
	heap.Init(lh)

	return &Cache{
		maxSize: size,
		meta:    &metaList{},
		lru:     lh,
	}
}

func (c *Cache) evict(toFree int) {
	freeSpace := c.maxSize - c.currentSize

	for freeSpace < toFree {
		var lru *lruObject
		lru = c.lru.Pop().(*lruObject)

		metaIndex, _ := c.search(lru.ptr.start, lru.ptr.end)
		freeSpace = freeSpace + c.meta.list[metaIndex].size
		// very fancy delete
		c.meta.list = append(c.meta.list[:metaIndex], c.meta.list[metaIndex+1:]...)
	}
}

// Put put
func (c *Cache) Put(start, end int, buffer []byte) error {
	// Locks cache
	c.lock.Lock()
	defer c.lock.Unlock()

	// If buffer is larger then cache maxSize, do not place into cache
	if len(buffer) > c.maxSize {
		return nil
	}

	// If buffer + current size of cache is greater then max, we need to evict items from cache
	if (len(buffer) + c.currentSize) > c.maxSize {
		c.evict(len(buffer))
	}

	newBuffer := make([]byte, len(buffer))
	copy(newBuffer, buffer)

	newMeta := &metaObject{
		start:  start,
		end:    end,
		buffer: newBuffer,
		size:   len(buffer),
	}

	newLru := &lruObject{
		epoch: time.Now().Unix(),
		ptr:   newMeta,
	}
	newMeta.lru = newLru

	c.lru.Push(newLru)
	fmt.Println("Placed something in cache: ")
	fmt.Println(newLru)
	c.meta.append(newMeta)
	c.currentSize = c.currentSize + newMeta.size

	return nil
}

// Get gets
func (c *Cache) Get(start, end int) ([]byte, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Attempt to retrieve index of meta index if found in metaList
	metaIndex, found := c.search(start, end)
	if !found {
		return nil, ErrCacheMiss
	}

	// Define buffer for return
	returnBuffer := make([]byte, end-start)

	// Obtain target metaObject, byte range index conversion from metaObj index to provided byte range
	targetMeta := c.meta.list[metaIndex]
	targetStartIndex := start - targetMeta.start
	targetEndIndex := targetStartIndex + (end - start)

	copy(returnBuffer, targetMeta.buffer[targetStartIndex:targetEndIndex])

	targetMeta.lru.epoch = time.Now().Unix()
	// sort.Sort(c.lru)

	fmt.Println("Got something from cache!")
	fmt.Println(targetMeta)
	fmt.Println(returnBuffer)
	return returnBuffer, nil
}

// Implementation of binary search, returns index of metaObj matching range provided
func (c *Cache) search(start, end int) (int, bool) {
	fmt.Println(len(c.meta.list))
	var mid int
	var found bool
	lower, upper := 0, len(c.meta.list)-1

	for lower <= upper {
		mid = (lower + upper)

		if c.meta.list[mid].start <= start && start < c.meta.list[mid].end {
			found = true
			break
		}

		if c.meta.list[mid].start < start {
			lower = mid + 1
		} else {
			upper = mid
		}
	}

	if found && end <= c.meta.list[mid].end {
		found = true
	}
	fmt.Println("Search:")
	fmt.Println(mid)
	fmt.Println(found)
	return mid, found
}
