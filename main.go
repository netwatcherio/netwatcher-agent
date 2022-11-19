package main

import (
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

	ApiUrl = os.Getenv("API_URL")
	if ApiUrl == "" {
		log.Fatal("You must insert the API URL")
	}

	var wg sync.WaitGroup
	log.Infof("Starting NetWatcher Agent...")
	log.Infof("Starting microsoft/ethr logging...")

	var agentConfig *agent_models.AgentConfig

	wg.Add(1)

	go func() {
		defer wg.Done()
		// Run forever if needed
		for {
			// A local reference to the agent config
			var conf *agent_models.AgentConfig
			// Attempt to pull the agent config
			conf, err = GetConfig()
			// If an error occurs, a message is logged to console and the loop repeats after one minute
			if err != nil {
				log.WithError(err).Warnf("Unable to fetch configuration, trying again in 1 minutes")
				time.Sleep(time.Minute)
				continue
			}
			// If there was no error, then the agent config is set
			agentConfig = conf

			log.Infof("Loaded %d agents", len(conf.AgentTargets))

		}
	}()

	wg.Wait()

	StartScheduler(agentConfig)

	wg.Wait()
}

func shutdown() {

	log.Fatal("Shutting down NetWatcher Agent...")

}
