package main

import (
	"encoding/json"
	"fmt"
	"github.com/netwatcherio/netwatcher-agent/api"
	"github.com/netwatcherio/netwatcher-agent/checks"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
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

	// initialize the data from api
	// todo make this a loop that checks periodically as well as handles the errors and retries
	data := client.Data()
	data.PIN = clientCfg.APIPassword
	data.ID = clientCfg.APIHost
	err := data.Initialize()
	if err != nil {
		log.Fatal(err)
		return
	}

	data.Checks = append(data.Checks, checks.CheckData{
		Type: "NETINFO",
	})

	for {
		if len(data.Checks) <= 0 {
			fmt.Println("no checks received, waiting for 2 minutes")
			time.Sleep(time.Minute * 2)
		} else {
			break
		}
	}

	dd := make(chan checks.CheckData)
	for _, d := range data.Checks {
		time.Sleep(time.Millisecond)
		switch strings.ToUpper(d.Type) {
		case "MTR":
			go func(checkData checks.CheckData) {
				for {
					fmt.Println("Running mtr test for ", checkData.Target, "...")
					mtr := checks.MtrResult{}
					err := mtr.Check(&checkData)
					if err != nil {
						fmt.Println(err)
					}
					fmt.Println("Sending data to the channel (MTR) for ", checkData.Target, "...")
					dd <- checkData
				}
			}(d)
			// todo push
			break
		case "RPERF":
			// if check says its a server, start a iperf server based on the bind and port provided in target
			if d.Server {
				/*func(checkData checks.CheckData) {
					rperf := checks.RPerfResults{}
					err := rperf.Check(&checkData)
					if err != nil {
						fmt.Println(err)
					}
				}(d)*/
			}
			go func(checkData checks.CheckData) {
				for {
					//todo
					//make this continue to run, however, make it check if the latest version of the check
					//data contains it, if not, then break out of this thread

					fmt.Println("Running rperf test for ", checkData.Target, "...")
					rperf := checks.RPerfResults{}
					err := rperf.Check(&checkData)
					if err != nil {
						fmt.Println(err)
						fmt.Println("something went wrong processing rperf... sleeping for 10 seconds")
						time.Sleep(time.Second * 10)
					}
					/*fmt.Println("something went wrong processing rperf... sleeping for 10 seconds")
					time.Sleep(time.Second * 10)*/

					fmt.Println("Sending data to the channel (RPERF) for ", checkData.Target, "...")
					dd <- checkData
				}
			}(d)
			break
		case "SPEEDTEST":
			go func(checkData checks.CheckData) {
				for {
					fmt.Println("Running speed test...")
					speedtest := checks.SpeedTest{}
					err := speedtest.Check()
					if err != nil {
						fmt.Println(err)
						return
					}
					dd <- checkData

					//todo make this onyl run once, because when it uploads to the server, it will disable it,
					//todo preventing it from being in the configuration after
					time.Sleep(time.Minute * 5)
				}
			}(d)
			break
		case "NETINFO":
			go func(checkData checks.CheckData) {
				for {
					fmt.Println("Checking networking information...")
					net := checks.NetResult{}
					err := net.Check(&checkData)
					if err != nil {
						fmt.Println(err)
					}
					dd <- checkData

					// todo make configurable??
					time.Sleep(time.Minute * 5)
				}
			}(d)
			break

		// todo other checks like port scans etc.

		default:
			fmt.Println("Unknown type of check...")
			break
		}
	}

	for {
		chand := <-dd
		//todo process data received from channel and add to queue
		marshal, err := json.Marshal(chand)
		if err != nil {
			return
		}
		print("\n\n\n--------------------------\n" + string(marshal) + "\n--------------------------\n\n\n")
	}
}

func shutdown() {
	log.Fatalf("Currently %d threads", runtime.NumGoroutine())
	log.Fatal("Shutting down NetWatcher Agent...")
}
