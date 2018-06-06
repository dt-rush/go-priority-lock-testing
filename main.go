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

var profile = flag.Bool("profile", false, "add this flag to cause a .pprof file to be produced (suppresses logging)")
var silent = flag.Bool("silent", false, "silence locks and unlock output")
var strategy = flag.String("strategy", "plock", "which strategy to use, plock-sleep, queued-plock-sleep, plock-gosched, or mutex")

const N_LOCKERS = 100
const N_PLOCKERS = 2

type PLock interface {
	Lock()
	Unlock()
	PLock()
	PUnlock()
}

func locker(
	pl PLock,
	stopFlag *atomic.Uint32,
	wg *sync.WaitGroup,
	lockCounter *atomic.Uint32) {

	for stopFlag.Load() == 0 {
		pl.Lock()
		lockCounter.Inc()
		Log(time.Now().UnixNano(), "lock ACQUIRED")
		time.Sleep(10 * time.Millisecond)
		Log(time.Now().UnixNano(), "lock RELEASING")
		pl.Unlock()
	}
	wg.Done()
}

func plocker(
	pl PLock,
	stopFlag *atomic.Uint32,
	wg *sync.WaitGroup,
	plockCounter *atomic.Uint32) {

	id := IDGEN()

	for stopFlag.Load() == 0 {
		time.Sleep(50 * time.Millisecond)
		Log(time.Now().UnixNano(), "p[%d] waiting", id)
		pl.PLock()
		plockCounter.Inc()
		Log(time.Now().UnixNano(), "p[%d] lock ACQUIRED", id)
		time.Sleep(10 * time.Millisecond)
		Log(time.Now().UnixNano(), "p[%d] lock RELEASING", id)
		pl.PUnlock()
	}
	wg.Done()
}

func main() {

	flag.Parse()

	var PL PLock
	switch *strategy {
	case "queued-plock-sleep":
		PL = &QueuedSleepPriorityLock{}
	case "plock-sleep":
		PL = &SleepPriorityLock{}
	case "plock-gosched":
		PL = &GoschedPriorityLock{}
	case "mutex":
		PL = &MutexPriorityLock{}
	default:
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
	var stopFlag atomic.Uint32
	var lockCounter atomic.Uint32
	var plockCounter atomic.Uint32
	var wg sync.WaitGroup
	// spawn lockers and plockers
	for i := 0; i < N_LOCKERS; i++ {
		wg.Add(1)
		go locker(PL, &stopFlag, &wg, &lockCounter)
	}
	for i := 0; i < N_PLOCKERS; i++ {
		wg.Add(1)
		go plocker(PL, &stopFlag, &wg, &plockCounter)
	}

	time.Sleep(10 * time.Second)
	stopFlag.Store(1)
	wg.Wait()

	fmt.Printf("%d locks,\t%d plocks\t(%.3f %% plock)",
		lockCounter.Load(), plockCounter.Load(),
		float32(plockCounter.Load())/
			float32(plockCounter.Load()+lockCounter.Load()))
}
