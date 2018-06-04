package main

import (
	"go.uber.org/atomic"
	"runtime"
)

type GoschedPriorityLock struct {
	lock     atomic.Int32
	pwaiting atomic.Uint32
}

// grab lock regular, deferring to priority lockers
func (gpl *GoschedPriorityLock) Lock() {
	var grabbed = false
	// loop until L grabbed
	for !grabbed {
		// sleep until L grabbed
		for !gpl.lock.CAS(OPEN, LOCKED) {
			runtime.Gosched()
		}
		// after grabbed, if PLOCK waiting, store 2 in lock
		if gpl.pwaiting.Load() > 0 {
			gpl.lock.Store(PRIORITY_RESERVED)
		} else {
			// else we're good, set grabbed = true
			// (hence we break loop and return)
			grabbed = true
		}
	}
}

func (gpl *GoschedPriorityLock) Unlock() {
	gpl.lock.Store(OPEN)
}

// grab lock with priority
// increment pWaiting, block until acquired open or reserved lock,
// finally decrement pwaiting
func (gpl *GoschedPriorityLock) PLock() {
	gpl.pwaiting.Inc()
	for !(gpl.lock.CAS(OPEN, LOCKED) ||
		gpl.lock.CAS(PRIORITY_RESERVED, LOCKED)) {
		runtime.Gosched()
	}
	gpl.pwaiting.Dec()
}

func (gpl *GoschedPriorityLock) PUnlock() {
	gpl.Unlock()
}
