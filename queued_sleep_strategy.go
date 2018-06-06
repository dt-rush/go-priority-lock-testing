package main

import (
	"go.uber.org/atomic"
	"time"
)

type QueuedSleepPriorityLock struct {
	queueTicket   atomic.Uint32
	dequeueTicket atomic.Uint32
	lock          atomic.Int32
	pWaiting      atomic.Uint32
}

// grab lock regular, deferring to priority lockers
func (qspl *QueuedSleepPriorityLock) Lock() {
	// grab ticket on entry
	queueTicket := qspl.queueTicket.Inc()
	for {
		// wait til we're at the front of the regular priority queue
		for qspl.dequeueTicket.Load() != queueTicket-1 {
			time.Sleep(time.Millisecond)
		}
		Log(time.Now().UnixNano(), "queueTicket %d at front", queueTicket)
		// try to grab lock from OPEN to LOCKED
		for !qspl.lock.CAS(OPEN, LOCKED) {
			time.Sleep(time.Millisecond)
		}
		Log(time.Now().UnixNano(), "queueTicket %d set QPLock -> LOCKED", queueTicket)
		// if, once locked, we find a priority locker waiting, set
		// to PRIORITY_RESERVED and continue AKA wait again
		// for CAS(OPEN -> LOCKED) (since nothing can come to front of queue
		// til this locker is unlocked, and if the user doesn't have logic
		// errors, that can't happen while we're still waiting to leave here
		if qspl.pWaiting.Load() > 0 {
			Log(time.Now().UnixNano(), "queueTicket %d locked but saw priority wait. Setting "+
				"QPLock -> PRIORITY_RESERVED", queueTicket)
			qspl.lock.Store(PRIORITY_RESERVED)
			continue
		} else {
			// we grabbed lock and no priority was waiting
			return
		}
	}
}

func (qspl *QueuedSleepPriorityLock) Unlock() {
	Log(time.Now().UnixNano(), "QPLock UNLOCK")
	if !qspl.lock.CAS(LOCKED, OPEN) {
		panic("Tried to unlock non-locked/priority-reserved QPLock, this " +
			"represents a logic error in your program")
	}
	qspl.dequeueTicket.Inc()
}

func (qspl *QueuedSleepPriorityLock) PUnlock() {
	Log(time.Now().UnixNano(), "QPLock PUNLOCK")
	qspl.lock.Store(OPEN)
}

// grab lock with priority
// increment pWaiting, block until acquired open or reserved lock,
// finally decrement pwaiting
func (qspl *QueuedSleepPriorityLock) PLock() {
	qspl.pWaiting.Inc()
	Log(time.Now().UnixNano(), "PLock() waiting...")
	for !(qspl.lock.CAS(OPEN, LOCKED) ||
		qspl.lock.CAS(PRIORITY_RESERVED, LOCKED)) {
		time.Sleep(time.Millisecond)
	}
	Log(time.Now().UnixNano(), "PLock() set QPLock -> LOCKED...")
	qspl.pWaiting.Dec()
}
