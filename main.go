package main

import (
	"encoding/json"
	"fmt"
	_ "github.com/joho/godotenv"
	"github.com/sagostin/netwatcher-agent/agent_models"
	"log"
	"sync"
)

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

func main() {

	var wg sync.WaitGroup

	fmt.Println("Starting NetWatcher Agent...")

	var t = []agent_models.MtrTarget{
		{
			Address: "1.1.1.1",
		},
	}

	for _, st := range t {
		wg.Add(1)
		go func() {
			defer wg.Done()
			CheckMTR(&st, 5)
		}()
	}

	wg.Wait()
	for s := range t {
		j, err := json.Marshal(t[s].Result)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Hop info %s %s\n", t[s].Address, string(j))
	}

	var t2 = []agent_models.IcmpTarget{
		{
			Address: "1.1.1.1",
		},
	}

	wg.Wait()

	for _, st := range t2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			CheckICMP(&st)
			fmt.Printf("Time for %s - %vms \n", st.Address, st.Result.ElapsedMilliseconds)
		}()
	}

	wg.Wait()

	wg.Add(1)
	fmt.Println("Running speed test...\n")
	go func() {
		defer wg.Done()
		speedInfo, err := RunSpeedTest()

		if err != nil {
			log.Fatalln(err)
		}

		fmt.Printf("Speed %fmbps - %fmbps \n%s %s\n", speedInfo.ULSpeed, speedInfo.DLSpeed, speedInfo.Server, speedInfo.Host)
	}()

	wg.Wait()

	wg.Add(1)
	fmt.Println("Getting network information...\n")
	go func() {
		defer wg.Done()
		networkInfo, err := CheckNetworkInfo()

		if err != nil {
			log.Fatalln(err)
		}

		fmt.Printf("Local Subnet: %s, Local Gateway: %s, ISP: %s, WAN IP: %s",
			networkInfo.LocalSubnet, networkInfo.DefaultGateway, networkInfo.InternetProvider, networkInfo.PublicAddress)
	}()

	wg.Wait()

}
