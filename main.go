package main

import (
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/netwatcherio/ethr"
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

	_, err = os.Stat("./config.conf")
	if errors.Is(err, os.ErrNotExist) {
		fmt.Println("file does not exist")
		// To start, here's how to dump a string (or just
		// bytes) into a file.
		_, err := os.Create("./config.conf")
		d1 := []byte("API_URL=*PUT URL HERE*\nPIN=*PUT PIN HERE*\nHASH=\n")
		err = os.WriteFile("./config.conf", d1, 0644)
		if err != nil {
			log.Fatal("Cannot create configuration.")
		}
	}

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
	if ApiUrl == "" {
		log.Fatal("You must insert the API URL")
	}

	var wg sync.WaitGroup
	log.Infof("Starting NetWatcher Agent...")
	log.Infof("Starting microsoft/ethr logging...")

	var nonCliConfig = &ethr.NonCliConfig{}
	ethr.RunEthr(false, nonCliConfig)
	var agentConfig *agent_models.AgentConfig

	ethrLogChan := <-ethr.LogChan
	go func() {
		for true {
			log.Warnf("%s", ethrLogChan)
		}
	}()

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
