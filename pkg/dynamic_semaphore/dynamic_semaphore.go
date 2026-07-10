package dynamic_semaphore

import "sync"

// DynamicSemaphore is a counting semaphore whose capacity can grow at
// runtime. Standard buffered channels can't do this (capacity is fixed at
// creation), so this implements acquire/release/grow manually with a
// mutex + FIFO of waiters.
type DynamicSemaphore struct {
	mu       sync.Mutex
	capacity int
	inUse    int
	waiters  []chan struct{}
}

func NewDynamicSemaphore(capacity int) *DynamicSemaphore {
	if capacity < 1 {
		capacity = 1
	}
	return &DynamicSemaphore{capacity: capacity}
}

// Acquire blocks until a slot is available.
func (s *DynamicSemaphore) Acquire() {
	s.mu.Lock()
	if s.inUse < s.capacity {
		s.inUse++
		s.mu.Unlock()
		return
	}

	wait := make(chan struct{})
	s.waiters = append(s.waiters, wait)
	s.mu.Unlock()

	<-wait
}

// Release frees a slot, handing it directly to the oldest waiter if any
// are queued.
func (s *DynamicSemaphore) Release() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.waiters) > 0 {
		next := s.waiters[0]
		s.waiters = s.waiters[1:]
		close(next) // hands the slot to this waiter; inUse stays the same
		return
	}

	s.inUse--
}

// Grow permanently increases capacity by n and immediately wakes up to n
// queued waiters. Used to let short maps skip ahead of long-running ones
// instead of waiting behind them for a fixed slot to free up.
func (s *DynamicSemaphore) Grow(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.capacity += n

	for n > 0 && len(s.waiters) > 0 {
		next := s.waiters[0]
		s.waiters = s.waiters[1:]
		s.inUse++
		close(next)
		n--
	}
}
