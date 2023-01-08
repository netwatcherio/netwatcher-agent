package main

import (
	"encoding/json"
	"fmt"
	"github.com/netwatcherio/netwatcher-agent/api"
	"github.com/netwatcherio/netwatcher-agent/checks"
	"github.com/netwatcherio/netwatcher-agent/workers"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
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

	workers.InitQueueWorker(checkDataCh, queueReq, apiClient)

	agentC := make(chan api.AgentCheck)

	var updateReceived = false

	// todo keep track of running tests once started, tests actively running cannot be changed only removed or *disabled
	go func(cd chan api.AgentCheck, received bool) {
		newCfg := true
		for {
			err := apiClient.Initialize(&apiRequest)
			if err != nil {
				fmt.Println(err)
			}

			b, err := json.Marshal(apiRequest.Data)
			if err != nil {
				log.Println(err)
			}
			log.Println("Update received: ", string(b))

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
			if !newCfg {
				received = true
			}

			if newCfg {
				newCfg = false
			}

			for i := range ce {
				cd <- ce[i]
			}
			time.Sleep(5 * time.Minute)
		}
	}(agentC, updateReceived)

	for {
		d := <-agentC
		agId, err := primitive.ObjectIDFromHex(apiRequest.ID)
		if err != nil {
			log.Fatal(err)
		}
		d.AgentID = agId

		time.Sleep(time.Millisecond)
		switch d.Type {
		case api.CtMtr:
			go func(ac api.AgentCheck) {
				for {
					if updateReceived {
						break
					}
					fmt.Println("Running mtr test for ", ac.Target, "...")
					mtr, err := checks.CheckMtr(&ac, false)
					if err != nil {
						fmt.Println(err)
					}

					m, err := json.Marshal(mtr)
					if err != nil {
						fmt.Print(err)
					}

					cD := api.CheckData{
						Target:    ac.Target,
						CheckID:   ac.ID,
						AgentID:   ac.AgentID,
						Triggered: mtr.Triggered,
						Result:    string(m),
						Type:      api.CtMtr,
					}

					fmt.Println("Sending apiClient to the channel (MTR) for ", ac.Interval, "...")
					checkDataCh <- cD
					fmt.Println("sleeping for " + strconv.Itoa(ac.Interval) + " minutes")
					time.Sleep(time.Duration(ac.Interval) * time.Minute)
				}
			}(d)
			// todo push
			continue
		case "RPERF":
			// if check says its a server, start a iperf server based on the bind and port provided in target
			if d.Server {
			}
			go func(ac api.AgentCheck) {
				for {
					if updateReceived {
						break
					}
					//todo
					//make this continue to run, however, make it check if the latest version of the check
					//apiClient contains it, if not, then break out of this thread

					fmt.Println("Running rperf test for ", ac.Target, "...")
					rperf := checks.RPerfResults{}
					err := rperf.Check(&ac)
					if err != nil {
						fmt.Println(err)
						fmt.Println("something went wrong processing rperf... sleeping for 10 seconds")
						time.Sleep(time.Second * 10)
					}

					m, err := json.Marshal(rperf)
					if err != nil {
						fmt.Print(err)
					}

					cD := api.CheckData{
						Target:  ac.Target,
						CheckID: ac.ID,
						AgentID: ac.AgentID,
						Result:  string(m),
						Type:    api.CtRperf,
					}

					fmt.Println("Sending apiClient to the channel (RPERF) for ", ac.Target, "...")
					checkDataCh <- cD
				}
			}(d)
			continue
		case "SPEEDTEST":
			go func(ac api.AgentCheck) {
				if ac.Pending {
					fmt.Println("Running speed test...")
					speedtest, err := checks.CheckSpeedTest(&ac)
					if err != nil {
						fmt.Println(err)
						return
					}

					m, err := json.Marshal(speedtest)
					if err != nil {
						fmt.Print(err)
					}

					cD := api.CheckData{
						Target:  ac.Target,
						CheckID: ac.ID,
						AgentID: ac.AgentID,
						Result:  string(m),
						Type:    api.CtSpeedtest,
					}

					checkDataCh <- cD

					//todo make this onyl run once, because when it uploads to the server, it will disable it,
					//todo preventing it from being in the configuration after
					//time.Sleep(time.Minute * 5)
					//}
				}
			}(d)
			continue
		case "NETINFO":
			go func(ac api.AgentCheck) {
				for {
					if updateReceived {
						break
					}

					fmt.Println("Checking networking information...")
					net, err := checks.CheckNet()
					if err != nil {
						fmt.Println(err)
					}

					m, err := json.Marshal(net)
					if err != nil {
						fmt.Print(err)
					}

					cD := api.CheckData{
						Target:  ac.Target,
						CheckID: ac.ID,
						AgentID: ac.AgentID,
						Result:  string(m),
						Type:    api.CtNetinfo,
					}

					checkDataCh <- cD

					// todo make configurable??
					time.Sleep(time.Minute * 10)
				}
			}(d)
			continue

		// todo other checks like port scans etc.

		default:
			fmt.Println("Unknown type of check...")
			continue
		}
	}
}

func shutdown() {
	log.Fatalf("Currently %d threads", runtime.NumGoroutine())
	log.Fatal("Shutting down NetWatcher Agent...")
}
