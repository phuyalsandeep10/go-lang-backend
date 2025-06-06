package cache

import "sync"

type Cache struct {
	store map[string]interface{}
	mu    sync.RWMutex
}

func NewCache() *Cache {
	return &Cache{
		store: make(map[string]interface{}),
	}
}

func (c *Cache) Set(key string, value interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = value
	return nil
}

func (c *Cache) GetAll() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[string]interface{})
	for k, v := range c.store {
		result[k] = v
	}
	return result
}
