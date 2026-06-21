package cache

import (
	"container/list"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"sharelink/internal/db"
)

type CacheItem struct {
	Key          string
	StatusCode   int
	Headers      http.Header
	Body         []byte
	CreatedAt    time.Time
	ExpiresAt    time.Time
	Size         int64
	HitCount     int
	LastAccessAt time.Time
}

type RAMCache struct {
	mu           sync.RWMutex
	items        map[string]*list.Element
	evictList    *list.List
	currentBytes int64
}

var (
	GlobalCache *RAMCache
	once        sync.Once
)

func GetGlobalCache() *RAMCache {
	once.Do(func() {
		GlobalCache = &RAMCache{
			items:     make(map[string]*list.Element),
			evictList: list.New(),
		}
	})
	return GlobalCache
}

// GetMaxMemoryLimitMB returns the global memory cache limit in MB from settings
func GetMaxMemoryLimitMB() int64 {
	if setting, found, err := db.FindGlobalSetting("global_cache_max_memory_mb"); err == nil && found {
		if val, err := strconv.ParseInt(setting.Value, 10, 64); err == nil {
			return val
		}
	}
	return 64 // default 64MB
}

// IsGlobalCacheEnabled returns if the global cache is enabled
func IsGlobalCacheEnabled() bool {
	if setting, found, err := db.FindGlobalSetting("global_cache_enabled"); err == nil && found {
		return setting.Value == "true" || setting.Value == "1"
	}
	return true // default enabled
}

func (c *RAMCache) Get(key string) (*CacheItem, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, exists := c.items[key]
	if !exists {
		return nil, false
	}

	item := element.Value.(*CacheItem)
	// Check if expired
	if time.Now().After(item.ExpiresAt) {
		// Remove expired item
		c.removeElement(element)
		return nil, false
	}

	item.HitCount++
	item.LastAccessAt = time.Now()
	c.evictList.MoveToFront(element)
	return item, true
}

func (c *RAMCache) Put(key string, statusCode int, headers http.Header, body []byte, ttl int) {
	if !IsGlobalCacheEnabled() {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if key already exists, update it
	if element, exists := c.items[key]; exists {
		c.removeElement(element)
	}

	itemSize := int64(len(body))
	maxBytes := GetMaxMemoryLimitMB() * 1024 * 1024

	// If the item itself exceeds the entire cache size, don't store it
	if itemSize > maxBytes {
		return
	}

	// Evict items until there is enough space
	for c.currentBytes+itemSize > maxBytes && c.evictList.Len() > 0 {
		c.evictOldest()
	}

	item := &CacheItem{
		Key:          key,
		StatusCode:   statusCode,
		Headers:      headers.Clone(),
		Body:         body,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(time.Duration(ttl) * time.Second),
		Size:         itemSize,
		LastAccessAt: time.Now(),
	}

	element := c.evictList.PushFront(item)
	c.items[key] = element
	c.currentBytes += itemSize
}

func (c *RAMCache) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, exists := c.items[key]; exists {
		c.removeElement(element)
	}
}

// ClearAll purges the entire RAM cache
func (c *RAMCache) ClearAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.evictList.Init()
	c.currentBytes = 0
}

// ClearLinkCache clears all cache keys belonging to the given prefix and slug
func (c *RAMCache) ClearLinkCache(prefix, slug string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	prefixSlugKey := prefix + "|" + slug + "|"
	for key, element := range c.items {
		if strings.HasPrefix(key, prefixSlugKey) {
			c.removeElement(element)
		}
	}
}

func (c *RAMCache) removeElement(e *list.Element) {
	c.evictList.Remove(e)
	item := e.Value.(*CacheItem)
	delete(c.items, item.Key)
	c.currentBytes -= item.Size
}

func (c *RAMCache) evictOldest() {
	element := c.evictList.Back()
	if element != nil {
		c.removeElement(element)
	}
}

// GetStatus returns the stats of the RAM cache
func (c *RAMCache) GetStatus() (int, int64, int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items), c.currentBytes, GetMaxMemoryLimitMB() * 1024 * 1024
}
