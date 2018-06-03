package main

import (
	"go.uber.org/atomic"
	"runtime"
	"sync"
	"time"
)

type GoschedPriorityLock struct {
	lock     atomic.Uint32
	pwaiting atomic.Uint32
}

// grab lock regular, deferring to priority lockers
func (gpl *GoschedPriorityLock) Lock() {
	var grabbed = false
	// loop until L grabbed
	for !grabbed {
		// sleep until L grabbed
		for !gpl.lock.CAS(0, 1) {
			runtime.Gosched()
		}
		// after grabbed, if PLOCK waiting, store 2 in lock
		if gpl.pwaiting.Load() > 0 {
			gpl.lock.Store(2)
		} else {
			// else we're good, set grabbed = true
			// (hence we break loop and return)
			grabbed = true
		}
	}
}

// unlock sets lock to 0
func (gpl *GoschedPriorityLock) Unlock() {
	gpl.lock.Store(0)
}

// grab lock with priority
func (gpl *GoschedPriorityLock) PLock(id int) {
	// increment pwaiting
	VerboseLog("p[%d] pwaiting = %d\n", id, gpl.pwaiting.Inc())
	// test both possible lock grabs until success
	for !(gpl.lock.CAS(0, 1) || gpl.lock.CAS(2, 1)) {
		runtime.Gosched()
	}
	// decrement pwaiting
	gpl.pwaiting.Dec()
}

func GoschedLocker(
	gpl *GoschedPriorityLock,
	stopFlag *atomic.Uint32,
	wg *sync.WaitGroup,
	lockCounter *atomic.Uint32) {

	for stopFlag.Load() == 0 {
		gpl.Lock()
		lockCounter.Inc()
		VerboseLog("lock ACQUIRED")
		time.Sleep(10 * time.Millisecond)
		VerboseLog("lock RELEASING")
		gpl.Unlock()
	}
	wg.Done()
}

func GoschedPLocker(id int,
	gpl *GoschedPriorityLock,
	stopFlag *atomic.Uint32,
	wg *sync.WaitGroup,
	plockCounter *atomic.Uint32) {

	for stopFlag.Load() == 0 {
		time.Sleep(50 * time.Millisecond)
		gpl.PLock(id)
		plockCounter.Inc()
		VerboseLog("p[%d] gplock ACQUIRED\n", id)
		time.Sleep(10 * time.Millisecond)
		VerboseLog("p[%d] gplock RELEASING\n", id)
		gpl.Unlock()
	}
	wg.Done()
}

func DoGoschedTest(
	stopFlag *atomic.Uint32,
	lockCounter *atomic.Uint32,
	plockCounter *atomic.Uint32) {

	gpl := GoschedPriorityLock{}
	wg := sync.WaitGroup{}

	for i := 0; i < N_LOCKERS; i++ {
		wg.Add(1)
		go GoschedLocker(&gpl, stopFlag, &wg, lockCounter)
	}
	for i := 0; i < N_PLOCKERS; i++ {
		wg.Add(1)
		go GoschedPLocker(i, &gpl, stopFlag, &wg, plockCounter)
	}
	time.Sleep(10 * time.Second)
	wg.Wait()
}
