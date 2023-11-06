package main

import (
	"fmt"
	"github.com/netwatcherio/netwatcher-agent/ws"
	"log"
	"os"
	"os/signal"
	"runtime"
)

func main() {
	fmt.Printf("Starting NetWatcher Agent...\n")

	loadConfig()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		for _ = range c {
			shutdown()
			return
		}
	}()

	wsH := &ws.WebSocketHandler{}
	wsH.InitWS(os.Getenv("HOST"), os.Getenv("HOST_WS"), os.Getenv("PIN"), os.Getenv("ID"))

	// so we need to, connect to the websocket, once connected, stay connected, but funnel the config pull
	// over to where it will handle managing the runners/workers who are the ones actually running the tests
	//

	// TODO
	// init websocket connection to backend and handle disconnects, etc.
	// mutex/lock configuration while reconnecting to backend
	// if backend has disconnected, write data to memory, and if continue to fail, save to raw json file

	select {}
	// try running this program twice or/and run the server's http://localhost:8080 to check the browser client as well.

	/*loadConfig()
	clientCfg := api.RestClientConfig{
		APIHost:     os.Getenv("HOST"),
		HTTPTimeout: 10 * time.Second,
		DialTimeout: 5 * time.Second,
		TLSTimeout:  5 * time.Second,
	}
	client := api.NewClient(clientCfg)

	// initialize the apiClient from api
	// todo make this a loop that checks periodically as well as handles the errors and retries
	apiClient := api.Data{
		RestClientConfig: client,
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

	select {}*/
}

func shutdown() {
	log.Fatalf("Currently %d threads", runtime.NumGoroutine())
	log.Fatal("Shutting down NetWatcher Agent...")
}
