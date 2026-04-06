package foostash

import (
	"sync"
	"time"
)

type cacheEntry struct {
	secrets map[string]string
	expires time.Time
}

type ttlCache struct {
	mu    sync.RWMutex
	ttl   time.Duration
	entry *cacheEntry
}

func newTTLCache(ttl time.Duration) *ttlCache {
	return &ttlCache{ttl: ttl}
}

func (c *ttlCache) get() (map[string]string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.entry == nil || time.Now().After(c.entry.expires) {
		return nil, false
	}
	return c.entry.secrets, true
}

func (c *ttlCache) set(secrets map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entry = &cacheEntry{
		secrets: secrets,
		expires: time.Now().Add(c.ttl),
	}
}

func (c *ttlCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entry = nil
}
