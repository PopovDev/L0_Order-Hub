package cache

import "sync"

type Cache struct {
    mu       sync.Mutex
    capacity int
    items    []item
}

type item struct {
    key   string
    value string
}

func New(capacity int) *Cache {
    return &Cache{
        capacity: capacity,
        items:    make([]item, 0, capacity),
    }
}

func (c *Cache) Set(key, value string) {
    c.mu.Lock()
    defer c.mu.Unlock()

    for i, it := range c.items {
        if it.key == key {
            c.items[i].value = value
            return
        }
    }

    if len(c.items) >= c.capacity {
        c.items = c.items[1:]
    }

    c.items = append(c.items, item{key: key, value: value})
}

func (c *Cache) Get(key string) (string, bool) {
    c.mu.Lock()
    defer c.mu.Unlock()

    for _, it := range c.items {
        if it.key == key {
            return it.value, true
        }
    }

    return "", false
}