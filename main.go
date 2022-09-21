package main

import (
	"github.com/joho/godotenv"
	"github.com/netwatcherio/netwatcher-agent/agent_models"
	log "github.com/sirupsen/logrus"
	"os"
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
	ApiUrl string
)

//todo implement nmap and iperf to main agents

func main() {
	var err error
	if err != nil {
		log.Fatal(err)
	}

	log.SetFormatter(&log.TextFormatter{})

	godotenv.Load("config.conf")

	/*signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	signal.Notify(signals, syscall.SIGKILL)
	go func() {
		s := <-signals
		log.Fatal("Received Signal: %s", s)
		shutdown()
		os.Exit(1)
	}()*/

	ApiUrl = os.Getenv("API_URL")

	var wg sync.WaitGroup
	log.Infof("Starting NetWatcher Agent...")

	var agentConfig *agent_models.AgentConfig

	/*agentConfig.TraceTargets = append(agentConfig.TraceTargets, "1.1.1.1")
	agentConfig.PingTargets = append(agentConfig.PingTargets, "1.1.1.1")*/

	wg.Add(1)
	go func() {
		defer wg.Done()
		var received = false
		for !received {
			conf, err := GetConfig()
			if err == nil {
				received = true
				agentConfig = conf
				log.Infof("Pulled configuration on start up")
			} else {
				log.Errorf("Unable to fetch configuration")
				time.Sleep(time.Minute)
			}
		}
	}()
	wg.Wait()
	StartScheduler(agentConfig)

	wg.Wait()
}

func shutdown() {
	log.Fatal("Shutting down NetWatcher Agent...")
}
