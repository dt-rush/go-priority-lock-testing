package main

import (
	"go.uber.org/atomic"
	"time"
)

type SleepPriorityLock struct {
	lock     atomic.Int32
	pwaiting atomic.Uint32
}

// grab lock regular, deferring to priority lockers
func (spl *SleepPriorityLock) Lock() {
	for {
		// grab lock
		for !spl.lock.CAS(OPEN, LOCKED) {
			time.Sleep(time.Millisecond)
		}
		// if pwaiting > 0, reserve the lock for priority lockers and continue
		// (only PLock or PLockMulti can CAS(PRIORITY_RESERVED, LOCKED or LOCKED_MULTI)
		if spl.pwaiting.Load() > 0 {
			spl.lock.Store(PRIORITY_RESERVED)
			continue
		} else {
			// we grabbed lock and no priority was waiting
			return
		}
	}
}

// unlock sets lock to 0
func (spl *SleepPriorityLock) Unlock() {
	spl.lock.Store(OPEN)
}

// grab lock with priority
// increment pWaiting, block until acquired open or reserved lock,
// finally decrement pwaiting
func (spl *SleepPriorityLock) PLock() {
	spl.pwaiting.Inc()
	for !(spl.lock.CAS(OPEN, LOCKED) ||
		spl.lock.CAS(PRIORITY_RESERVED, LOCKED)) {
		time.Sleep(time.Millisecond)
	}
	spl.pwaiting.Dec()
}

func (spl *SleepPriorityLock) PUnlock() {
	spl.Unlock()
}
