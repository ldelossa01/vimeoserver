package cache

import (
	"container/heap"
	"sort"
	"sync"
	"time"
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

// MemCache is a cache
type MemCache struct {
	maxSize     int
	currentSize int
	lru         *lruHeap
	lock        sync.Mutex
	sourceMap   map[string]*metaList
}

type metaObject struct {
	start     int
	end       int
	lru       *lruObject
	size      int
	buffer    []byte
	sourceURL string
}

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

func (c *MemCache) evict(toFree int) {
	freeSpace := c.maxSize - c.currentSize

	for freeSpace < toFree {
		var lru *lruObject
		lru = c.lru.Pop().(*lruObject)
		targetMetaList := c.sourceMap[lru.ptr.sourceURL]

		metaIndex, _ := c.search(lru.ptr.start, lru.ptr.end, lru.ptr.sourceURL)
		freeSpace = freeSpace + targetMetaList.list[metaIndex].size
		// very fancy delete
		targetMetaList.list = append(targetMetaList.list[:metaIndex], targetMetaList.list[metaIndex+1:]...)
	}
}

// Put put
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

	newBuffer := make([]byte, len(buffer))
	copy(newBuffer, buffer)

	newMeta := &metaObject{
		start:     start,
		end:       end,
		buffer:    newBuffer,
		size:      len(buffer),
		sourceURL: sourceURL,
	}

	newLru := &lruObject{
		epoch: time.Now().Unix(),
		ptr:   newMeta,
	}
	newMeta.lru = newLru

	if metaListObj, ok := c.sourceMap[sourceURL]; ok {
		metaListObj.append(newMeta)
	} else {
		newMetaList := &metaList{}
		c.sourceMap[sourceURL] = newMetaList
		c.sourceMap[sourceURL].append(newMeta)
	}

	c.lru.Push(newLru)
	c.currentSize = c.currentSize + newMeta.size

	return nil
}

// Get gets
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

	copy(returnBuffer, targetMeta.buffer[targetStartIndex:targetEndIndex])

	targetMeta.lru.epoch = time.Now().Unix()

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
