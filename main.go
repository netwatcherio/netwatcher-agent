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

	godotenv.Load()

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

/*
https://stackoverflow.com/questions/51717409/is-there-any-way-to-sign-the-windows-executables-generated-by-the-go-compiler How to Sign Windows Applications to prevent AV from detecting it...??
AVs don't like GoLang because the weird binary structure
*/

//import "os"

/*
Obkio is using WebSockets to control information, instead the device stores the information
for MTR and such. We want to store it on the server.

*/

/*

TODO

Use dotenv for local configuration, and save to file with hash

agent can poll custom destinations (http, icmp, etc., mainly simple checks)
agent can run mtr tests to custom destinations at a set interval
agent can run speed tests to remote sources (ookla? or custom??)
agent cam poll destinations or other agents
agent communicates to frontend/backend using web sockets
agent grabs configuration using http
snmp component

*/
