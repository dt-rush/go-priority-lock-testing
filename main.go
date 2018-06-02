package main

import (
	"flag"
	"fmt"
	"go.uber.org/atomic"
	"log"
	"os"
	"runtime/pprof"
	"sync"
	"time"
)

var profile = flag.Bool("profile", false, "which strategy to use, plock or mutex")
var strategy = flag.String("strategy", "plock", "which strategy to use, plock or mutex")

func Log(s string, args ...interface{}) {
	if !(*profile) {
		if len(args) == 0 {
			fmt.Println(s)
		} else {
			fmt.Printf(s, args...)
		}
	}
}

const N_LOCKERS = 100
const N_PLOCKERS = 2

type PriorityLock struct {
	lock     atomic.Uint32
	pwaiting atomic.Uint32
}

// grab lock regular, deferring to priority lockers
func (pl *PriorityLock) Lock() {
	var grabbed = false
	// loop until L grabbed
	for !grabbed {
		// sleep until L grabbed
		for !pl.lock.CAS(0, 1) {
			time.Sleep(time.Millisecond)
		}
		// after grabbed, if PLOCK waiting, store 2 in lock
		if pl.pwaiting.Load() > 0 {
			pl.lock.Store(2)
		} else {
			// else we're good, set grabbed = true
			// (hence we break loop and return)
			grabbed = true
		}
	}
}

// unlock sets lock to 0
func (pl *PriorityLock) Unlock() {
	pl.lock.Store(0)
}

// grab lock with priority
func (pl *PriorityLock) PLock(id int) {
	// increment pwaiting
	Log("p[%d] pwaiting = %d\n", id, pl.pwaiting.Inc())
	// test both possible lock grabs until success
	for !(pl.lock.CAS(0, 1) || pl.lock.CAS(2, 1)) {
		time.Sleep(time.Millisecond)
	}
	// decrement pwaiting
	pl.pwaiting.Dec()
}

func Locker(
	pl *PriorityLock,
	stopFlag *atomic.Uint32,
	wg *sync.WaitGroup) {

	for stopFlag.Load() == 0 {
		pl.Lock()
		Log("lock ACQUIRED")
		time.Sleep(10 * time.Millisecond)
		Log("lock RELEASING")
		pl.Unlock()
	}
	wg.Done()
}

func MutexLocker(
	m *sync.RWMutex,
	stopFlag *atomic.Uint32,
	wg *sync.WaitGroup) {

	for stopFlag.Load() == 0 {
		m.RLock()
		Log("lock ACQUIRED")
		time.Sleep(10 * time.Millisecond)
		Log("lock RELEASING")
		m.RUnlock()
	}
	wg.Done()
}

func PLocker(id int,
	pl *PriorityLock,
	stopFlag *atomic.Uint32,
	wg *sync.WaitGroup) {

	for stopFlag.Load() == 0 {
		time.Sleep(50 * time.Millisecond)
		pl.PLock(id)
		Log("p[%d] plock ACQUIRED\n", id)
		time.Sleep(10 * time.Millisecond)
		Log("p[%d] plock RELEASING\n", id)
		pl.Unlock()
	}
	wg.Done()
}

func MutexPLocker(id int,
	m *sync.RWMutex,
	stopFlag *atomic.Uint32,
	wg *sync.WaitGroup) {

	for stopFlag.Load() == 0 {
		time.Sleep(50 * time.Millisecond)
		m.Lock()
		Log("p[%d] plock ACQUIRED\n", id)
		time.Sleep(10 * time.Millisecond)
		Log("p[%d] plock RELEASING\n", id)
		m.Unlock()
	}
	wg.Done()
}

func DoPLockTest() {
	pl := PriorityLock{}
	stopFlag := atomic.NewUint32(0)
	wg := sync.WaitGroup{}

	for i := 0; i < N_LOCKERS; i++ {
		wg.Add(1)
		go Locker(&pl, stopFlag, &wg)
	}
	for i := 0; i < N_PLOCKERS; i++ {
		wg.Add(1)
		go PLocker(i, &pl, stopFlag, &wg)
	}
	time.Sleep(10 * time.Second)
	stopFlag.Store(1)
	wg.Wait()
}

func DoRWMutexTest() {
	m := sync.RWMutex{}
	stopFlag := atomic.NewUint32(0)
	wg := sync.WaitGroup{}

	for i := 0; i < N_LOCKERS; i++ {
		wg.Add(1)
		go MutexLocker(&m, stopFlag, &wg)
	}
	for i := 0; i < N_PLOCKERS; i++ {
		wg.Add(1)
		go MutexPLocker(i, &m, stopFlag, &wg)
	}
	time.Sleep(10 * time.Second)
	stopFlag.Store(1)
	wg.Wait()
}

func main() {

	flag.Parse()

	var TestF func()
	if *strategy == "plock" {
		TestF = DoPLockTest
	} else if *strategy == "mutex" {
		TestF = DoRWMutexTest
	} else {
		fmt.Println("strategy must be either mutex or plock")
		flag.Usage()
		os.Exit(1)
	}
	if *profile {
		f, err := os.Create(fmt.Sprintf("%s.pprof", *strategy))
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	TestF()
}
