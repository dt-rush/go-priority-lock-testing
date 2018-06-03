package main

import (
	"go.uber.org/atomic"
	"sync"
	"time"
)

func RWMutexLocker(
	m *sync.RWMutex,
	stopFlag *atomic.Uint32,
	wg *sync.WaitGroup,
	lockCounter *atomic.Uint32) {

	for stopFlag.Load() == 0 {
		m.RLock()
		VerboseLog("lock ACQUIRED")
		lockCounter.Inc()
		time.Sleep(10 * time.Millisecond)
		VerboseLog("lock RELEASING")
		m.RUnlock()
	}
	wg.Done()
}

func RWMutexPLocker(id int,
	m *sync.RWMutex,
	stopFlag *atomic.Uint32,
	wg *sync.WaitGroup,
	plockCounter *atomic.Uint32) {

	for stopFlag.Load() == 0 {
		time.Sleep(50 * time.Millisecond)
		m.Lock()
		plockCounter.Inc()
		VerboseLog("p[%d] plock ACQUIRED\n", id)
		time.Sleep(10 * time.Millisecond)
		VerboseLog("p[%d] plock RELEASING\n", id)
		m.Unlock()
	}
	wg.Done()
}

func DoRWMutexTest(
	stopFlag *atomic.Uint32,
	lockCounter *atomic.Uint32,
	plockCounter *atomic.Uint32) {

	m := sync.RWMutex{}
	wg := sync.WaitGroup{}

	for i := 0; i < N_LOCKERS; i++ {
		wg.Add(1)
		go RWMutexLocker(&m, stopFlag, &wg, lockCounter)
	}
	for i := 0; i < N_PLOCKERS; i++ {
		wg.Add(1)
		go RWMutexPLocker(i, &m, stopFlag, &wg, plockCounter)
	}
	time.Sleep(10 * time.Second)
	wg.Wait()
}
