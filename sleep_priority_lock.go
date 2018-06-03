package main

import (
	"go.uber.org/atomic"
	"sync"
	"time"
)

type SleepPriorityLock struct {
	lock     atomic.Uint32
	pwaiting atomic.Uint32
}

// grab lock regular, deferring to priority lockers
func (spl *SleepPriorityLock) Lock() {
	var grabbed = false
	// loop until L grabbed
	for !grabbed {
		// sleep until L grabbed
		for !spl.lock.CAS(0, 1) {
			time.Sleep(time.Millisecond)
		}
		// after grabbed, if PLOCK waiting, store 2 in lock
		if spl.pwaiting.Load() > 0 {
			spl.lock.Store(2)
		} else {
			// else we're good, set grabbed = true
			// (hence we break loop and return)
			grabbed = true
		}
	}
}

// unlock sets lock to 0
func (spl *SleepPriorityLock) Unlock() {
	spl.lock.Store(0)
}

// grab lock with priority
func (spl *SleepPriorityLock) PLock(id int) {
	// increment pwaiting
	VerboseLog("p[%d] pwaiting = %d\n", id, spl.pwaiting.Inc())
	// test both possible lock grabs until success
	for !(spl.lock.CAS(0, 1) || spl.lock.CAS(2, 1)) {
		time.Sleep(time.Millisecond)
	}
	// decrement pwaiting
	spl.pwaiting.Dec()
}

func SleepLocker(
	spl *SleepPriorityLock,
	stopFlag *atomic.Uint32,
	wg *sync.WaitGroup,
	lockCounter *atomic.Uint32) {

	for stopFlag.Load() == 0 {
		spl.Lock()
		lockCounter.Inc()
		VerboseLog("lock ACQUIRED")
		time.Sleep(10 * time.Millisecond)
		VerboseLog("lock RELEASING")
		spl.Unlock()
	}
	wg.Done()
}

func SleepPLocker(id int,
	spl *SleepPriorityLock,
	stopFlag *atomic.Uint32,
	wg *sync.WaitGroup,
	plockCounter *atomic.Uint32) {

	for stopFlag.Load() == 0 {
		time.Sleep(50 * time.Millisecond)
		spl.PLock(id)
		plockCounter.Inc()
		VerboseLog("p[%d] splock ACQUIRED\n", id)
		time.Sleep(10 * time.Millisecond)
		VerboseLog("p[%d] splock RELEASING\n", id)
		spl.Unlock()
	}
	wg.Done()
}

func DoSleepTest(
	stopFlag *atomic.Uint32,
	lockCounter *atomic.Uint32,
	plockCounter *atomic.Uint32) {

	spl := SleepPriorityLock{}
	wg := sync.WaitGroup{}

	for i := 0; i < N_LOCKERS; i++ {
		wg.Add(1)
		go SleepLocker(&spl, stopFlag, &wg, lockCounter)
	}
	for i := 0; i < N_PLOCKERS; i++ {
		wg.Add(1)
		go SleepPLocker(i, &spl, stopFlag, &wg, plockCounter)
	}
	time.Sleep(10 * time.Second)
	wg.Wait()
}
