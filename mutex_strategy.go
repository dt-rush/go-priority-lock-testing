package main

import (
	"go.uber.org/atomic"
	"sync"
	"time"
)

type MutexPriorityLock struct {
	lock     sync.Mutex
	pwaiting atomic.Uint32
}

// grab lock regular, deferring to priority lockers
func (mpl *MutexPriorityLock) Lock() {
	var grabbed = false
	for !grabbed {
		mpl.lock.Lock()
		if mpl.pwaiting.Load() > 0 {
			mpl.lock.Unlock()
			time.Sleep(time.Millisecond)
		} else {
			grabbed = true
		}
	}
}

func (mpl *MutexPriorityLock) Unlock() {
	mpl.lock.Unlock()
}

func (mpl *MutexPriorityLock) PLock() {
	mpl.pwaiting.Inc()
	mpl.lock.Lock()
	mpl.pwaiting.Dec()
}

func (mpl *MutexPriorityLock) PUnlock() {
	mpl.Unlock()
}
