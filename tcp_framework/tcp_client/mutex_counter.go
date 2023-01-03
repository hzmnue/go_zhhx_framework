package tcp_client

import "sync"

type MutexCounter[T uint8 | uint16 | uint32 | uint64] struct {
	mu     sync.RWMutex
	number T
}

func (c *MutexCounter[T]) Add(num T) T {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.number = c.number + num
	return c.number
}

func (c *MutexCounter[T]) Read() T {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.number
}
