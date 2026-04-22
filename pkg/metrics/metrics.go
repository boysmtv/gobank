package metrics

import "sync"

type Collector struct {
	mu       sync.RWMutex
	counters map[string]int64
}

func New() *Collector {
	return &Collector{
		counters: make(map[string]int64),
	}
}

func (c *Collector) Increment(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.counters[name]++
}

func (c *Collector) Snapshot() map[string]int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]int64, len(c.counters))
	for key, value := range c.counters {
		result[key] = value
	}

	return result
}
