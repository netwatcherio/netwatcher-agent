package main

import (
	"encoding/json"
	"fmt"
	"github.com/netwatcherio/netwatcher-agent/api"
	"github.com/netwatcherio/netwatcher-agent/workers"
	"log"
	"os"
	"os/signal"
	"runtime"
	"time"
)

func main() {
	fmt.Printf("Starting NetWatcher Agent...\n")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		for _ = range c {
			shutdown()
			return
		}
	}()

	setup()
	clientCfg := api.ClientConfig{
		APIHost:     os.Getenv("HOST"),
		HTTPTimeout: 10 * time.Second,
		DialTimeout: 5 * time.Second,
		TLSTimeout:  5 * time.Second,
	}
	client := api.NewClient(clientCfg)

	// initialize the apiClient from api
	// todo make this a loop that checks periodically as well as handles the errors and retries
	apiClient := api.Data{
		Client: client,
	}

	apiRequest := api.ApiRequest{ID: os.Getenv("ID"), PIN: os.Getenv("PIN")}

	// init queue
	queueReq := api.ApiRequest{
		PIN:   apiRequest.PIN,
		ID:    apiRequest.ID,
		Data:  nil,
		Error: "",
	}
	checkDataCh := make(chan api.CheckData)
	agentC := make(chan []api.AgentCheck)

	workers.InitQueueWorker(checkDataCh, queueReq, clientCfg)
	workers.InitCheckWorker(agentC, checkDataCh)

	var updateReceived = false

	// todo keep track of running tests once started, tests actively running cannot be changed only removed or *disabled
	go func(cd chan []api.AgentCheck, received bool) {
		for {
			err := apiClient.Initialize(&apiRequest)
			if err != nil {
				fmt.Println(err)
			}

			b, err := json.Marshal(apiRequest.Data)
			if err != nil {
				log.Println(err)
			}
			log.Println("Config received: ", string(b))

			var ce []api.AgentCheck

			err = json.Unmarshal(b, &ce)
			if err != nil {
				log.Println(err)
			}

			if len(ce) <= 0 {
				fmt.Println("no checks received, waiting for 10 seconds")
				time.Sleep(time.Second * 10)
				continue
			}

			cd <- ce

			time.Sleep(10 * time.Second)
		}
	}(agentC, updateReceived)

	select {}
}

func shutdown() {
	log.Fatalf("Currently %d threads", runtime.NumGoroutine())
	log.Fatal("Shutting down NetWatcher Agent...")
}
