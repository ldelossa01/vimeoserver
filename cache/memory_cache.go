package cache

import (
	"container/heap"
	"sort"
	"sync"
	"time"
)

// Implements list to hold meta objects, implements Sort interface
type metaList struct {
	list []*metaObject
}

func (m *metaList) Len() int           { return len(m.list) }
func (m *metaList) Swap(i, j int)      { m.list[i], m.list[j] = m.list[j], m.list[i] }
func (m *metaList) Less(i, j int) bool { return m.list[i].start < m.list[j].start }
func (m *metaList) append(newEle *metaObject) {
	m.list = append(m.list, newEle)
	sort.Sort(m)
}

// Implements min-heap for cache eviction, implements heap interface with type assertion in Push
type lruHeap []*lruObject

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

// MemCache is an in memory LRU-Cache
type MemCache struct {
	maxSize     int
	currentSize int
	lru         *lruHeap
	lock        sync.Mutex
	sourceMap   map[string]*metaList
}

// Implements object for holding meta-data and byte array for cache entries
type metaObject struct {
	start     int
	end       int
	lru       *lruObject
	size      int
	buffer    []byte
	sourceURL string
}

// Implements object for lruHeap, has pointer to metaObject, and lruHeap is sorted on key: epoch
type lruObject struct {
	epoch int64
	ptr   *metaObject
}

// NewMemCache cache factory
func NewMemCache(sizeMb int) *MemCache {
	size := sizeMb * 1000000
	lh := &lruHeap{}
	heap.Init(lh)

	return &MemCache{
		maxSize:   size,
		lru:       lh,
		sourceMap: make(map[string]*metaList),
	}
}

// Private function for handling evictions of objects from cache
func (c *MemCache) evict(toFree int) {
	freeSpace := c.maxSize - c.currentSize

	for freeSpace < toFree {
		var lru *lruObject
		lru = c.lru.Pop().(*lruObject)
		targetMetaList := c.sourceMap[lru.ptr.sourceURL]

		metaIndex, _ := c.search(lru.ptr.start, lru.ptr.end, lru.ptr.sourceURL)
		freeSpace = freeSpace + targetMetaList.list[metaIndex].size

		// Delete item from list
		targetMetaList.list = append(targetMetaList.list[:metaIndex], targetMetaList.list[metaIndex+1:]...)
	}
}

// Put places item into cache
func (c *MemCache) Put(start, end int, buffer []byte, sourceURL string) error {
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

	// Create new buffer for metaObj creation
	newBuffer := make([]byte, len(buffer))
	copy(newBuffer, buffer)

	// Create metObj
	newMeta := &metaObject{
		start:     start,
		end:       end,
		buffer:    newBuffer, // Holds bytes within specified byte ranges
		size:      len(buffer),
		sourceURL: sourceURL,
	}

	// Creates new lruObj with timestamp
	newLru := &lruObject{
		epoch: time.Now().Unix(),
		ptr:   newMeta,
	}

	// Bind newLru to newMeta
	newMeta.lru = newLru

	// Do we have a hash for the sourceURL? If yes, append to found metaList obj
	// if not create new metaList and create hash key for new sourceURL.
	// Append newMeta to new metaListObj
	if metaListObj, ok := c.sourceMap[sourceURL]; ok {
		metaListObj.append(newMeta)
	} else {
		newMetaList := &metaList{}
		c.sourceMap[sourceURL] = newMetaList
		c.sourceMap[sourceURL].append(newMeta)
	}

	// Push newLru onto heap
	c.lru.Push(newLru)
	// Update cache's current size
	c.currentSize = c.currentSize + newMeta.size

	return nil
}

// Get retreives items from the cache
func (c *MemCache) Get(start, end int, sourceURL string) ([]byte, error) {

	var targetMetaList *metaList

	c.lock.Lock()
	defer c.lock.Unlock()

	// Test to see if sourceURL is in sourceMap
	if t, ok := c.sourceMap[sourceURL]; ok {
		targetMetaList = t
	} else {
		return nil, ErrCacheMiss
	}

	// Attempt to retrieve index of meta index if found in metaList
	metaIndex, found := c.search(start, end, sourceURL)
	if !found {
		return nil, ErrCacheMiss
	}

	// Define buffer for return
	returnBuffer := make([]byte, end-start)

	// Obtain target metaObject, byte range index conversion from metaObj index to provided byte range
	targetMeta := targetMetaList.list[metaIndex]
	targetStartIndex := start - targetMeta.start
	targetEndIndex := targetStartIndex + (end - start)

	// Copy buffer for return
	copy(returnBuffer, targetMeta.buffer[targetStartIndex:targetEndIndex])

	// Update metaObj's lru epoch timestamp
	targetMeta.lru.epoch = time.Now().Unix()
	// Fix heap ordering after inclusion
	heap.Fix(c.lru, c.lru.Len()-1)

	return returnBuffer, nil
}

// Implementation of binary search, returns index of metaObj matching range provided
func (c *MemCache) search(start, end int, sourceURL string) (int, bool) {
	var mid int
	var found bool
	var targetMetaList *metaList

	if _, ok := c.sourceMap[sourceURL]; !ok {
		found = false
		return 0, found
	}
	targetMetaList = c.sourceMap[sourceURL]

	lower, upper := 0, len(targetMetaList.list)-1

	for lower <= upper {
		mid = (lower + upper) / 2

		if targetMetaList.list[mid].start <= start && start < targetMetaList.list[mid].end {
			found = true
			break
		}

		if targetMetaList.list[mid].start < start {
			lower = mid + 1
		} else {
			upper = mid - 1
		}
	}

	if found && end <= targetMetaList.list[mid].end {
		found = true
	}

	return mid, found
}
