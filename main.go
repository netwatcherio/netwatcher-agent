package main

import (
	"encoding/json"
	"fmt"
	"github.com/netwatcherio/netwatcher-agent/api"
	"github.com/netwatcherio/netwatcher-agent/checks"
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
	clientCfg := api.NewClientConfig()
	client := api.NewClient(clientCfg)

	// initialize the apiClient from api
	// todo make this a loop that checks periodically as well as handles the errors and retries
	apiClient := api.Data{
		Client: client,
	}

	apiRequest := api.ApiRequest{ID: os.Getenv("ID"), PIN: os.Getenv("PIN")}
	var checkData []api.AgentCheck

	for {
		err := apiClient.Initialize(&apiRequest)
		if err != nil {
			fmt.Println(err)
		}

		/*apiClient.Checks = append(apiClient.Checks, checks.CheckData{Type: "MTR", Target: "vultr1.gw.dec0de.xyz", Duration: 5})
		apiClient.Checks = append(apiClient.Checks, checks.CheckData{Type: "MTR", Target: "ovh1.gw.dec0de.xyz", Duration: 5})
		apiClient.Checks = append(apiClient.Checks, checks.CheckData{Type: "SPEEDTEST"})*/

		b, err := json.Marshal(apiRequest.Data)
		if err != nil {
			log.Println(err)
		}
		log.Println(string(b))

		err = json.Unmarshal(b, &checkData)
		if err != nil {
			log.Println(err)
		}

		if len(checkData) <= 0 {
			fmt.Println("no checks received, waiting for 10 seconds")
			time.Sleep(time.Second * 10)
		} else {
			break
		}
	}

	dd := make(chan api.CheckData)
	for _, d := range checkData {
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
					dd <- cD
					fmt.Println("sleeping for " + strconv.Itoa(ac.Interval) + " minutes")
					time.Sleep(time.Duration(ac.Interval) * time.Minute)
				}
			}(d)
			// todo push
			break
		case "RPERF":
			// if check says its a server, start a iperf server based on the bind and port provided in target
			if d.Server {
			}
			go func(ac api.AgentCheck) {
				for {
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
					dd <- cD
				}
			}(d)
			break
		case "SPEEDTEST":
			go func(ac api.AgentCheck) {
				//for {
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

					dd <- cD

					//todo make this onyl run once, because when it uploads to the server, it will disable it,
					//todo preventing it from being in the configuration after
					//time.Sleep(time.Minute * 5)
					//}
				}
			}(d)
			break
		case "NETINFO":
			go func(ac api.AgentCheck) {
				for {
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

					dd <- cD

					// todo make configurable??
					time.Sleep(time.Minute * 10)
				}
			}(d)
			break

		// todo other checks like port scans etc.

		default:
			fmt.Println("Unknown type of check...")
			break
		}
	}

	// init queue
	queue := api.ApiRequest{
		PIN:   apiRequest.PIN,
		ID:    apiRequest.ID,
		Data:  nil,
		Error: "",
	}

	var queueData []api.CheckData

	for {
		cD := <-dd
		queueData = append(queueData, cD)
		// make new object??

		m, err := json.Marshal(queueData)
		queue.Data = string(m)

		print("\n\n\n--------------------------\n" + string(m) + "\n--------------------------\n\n\n")

		err = apiClient.Push(&queue)
		if err != nil {
			// handle error on push and save queue for next time??
			log.Println("unable to push apiClient, keeping queue and waiting...")
			continue
		}
		queueData = nil
		queue.Data = nil
	}
}

func shutdown() {
	log.Fatalf("Currently %d threads", runtime.NumGoroutine())
	log.Fatal("Shutting down NetWatcher Agent...")
}
