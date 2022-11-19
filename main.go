package main

import (
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"time"
)

var (
	timeout        = 800 * time.Millisecond
	interval       = 100 * time.Millisecond
	hopSleep       = time.Nanosecond
	maxHops        = 64
	maxUnknownHops = 10
	ringBufferSize = 50
	ptrLookup      = false
	srcAddr        = ""
	ttl            = 60
)

var (
	ApiUrl   string
	OsDetect string
)

// todo implement nmap and iperf to main agents

func main() {

	log.SetFormatter(&log.TextFormatter{})

	err := setup()
	if err != nil {
		log.WithError(err).Fatal("An unexpected error occurred while configuring the agent")
		return
	}

	/*signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	signal.Notify(signals, syscall.SIGKILL)
	go func() {
		s := <-signals
		log.Fatal("Received Signal: %s", s)
		shutdown()
		os.Exit(1)
	}()*/

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		for _ = range c {
			shutdown()
			return
		}
	}()

	OsDetect = runtime.GOOS
	ApiUrl = os.Getenv("API_URL")
	if ApiUrl == "" {
		log.Fatal("You must insert the API URL")
	}

	var wg sync.WaitGroup
	log.Infof("Starting NetWatcher Agent...")
	log.Infof("Starting microsoft/ethr logging...")
	wg.Wait()

	StartScheduler()

}

func shutdown() {
	log.Fatalf("Currently %d threads", runtime.NumGoroutine())
	log.Fatal("Shutting down NetWatcher Agent...")
}
