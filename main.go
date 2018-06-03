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
var verbose = flag.Bool("verbose", false, "show locks and unlocks")
var strategy = flag.String("strategy", "plock", "which strategy to use, plock-sleep, plock-gosched, or mutex")

func Log(s string, args ...interface{}) {
	if !(*profile) {
		if len(args) == 0 {
			fmt.Println(s)
		} else {
			fmt.Printf(s, args...)
		}
	}
}

func VerboseLog(s string, args ...interface{}) {
	if *verbose {
		Log(s, args...)
	}
}

const N_LOCKERS = 100
const N_PLOCKERS = 2

type TestFunc func(
	stopFlag *atomic.Uint32,
	lockCounter *atomic.Uint32,
	plockCounter *atomic.Uint32)

func main() {

	flag.Parse()

	var TestF TestFunc
	if *strategy == "plock-sleep" {
		TestF = DoSleepTest
	} else if *strategy == "plock-gosched" {
		TestF = DoGoschedTest
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
	var stopFlag atomic.Uint32
	var lockCounter atomic.Uint32
	var plockCounter atomic.Uint32
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		TestF(&stopFlag, &lockCounter, &plockCounter)
		Log("%d locks,\t%d plocks\t(%.3f %% plock)\n",
			lockCounter.Load(), plockCounter.Load(),
			float32(plockCounter.Load())/
				float32(plockCounter.Load()+lockCounter.Load()))
		wg.Done()
	}()
	time.Sleep(10 * time.Second)
	stopFlag.Store(1)
	wg.Wait()
}
