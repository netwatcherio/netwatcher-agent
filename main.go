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

	// ICMP TARGET EXAMPLE

	var t2 = []*agent_models.IcmpTarget{
		{
			Address: "1.1.1.1",
		},
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		TestIcmpTargets(t2, 15, 2)

		for _, st := range t2 {
			j, err := json.Marshal(st)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%s\n", string(j))
		}
	}()

	// MTR TARGET EXAMPLE

	var t = []*agent_models.MtrTarget{
		{
			Address: "1.1.1.1",
		},
		{
			Address: "8.8.8.8",
		},
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		TestMtrTargets(t)
		for _, st := range t {
			j, err := json.Marshal(st)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%s\n", string(j))
		}
	}()

	// SPEED TEST
	wg.Add(1)
	go func() {
		defer wg.Done()
		speedInfo, err := RunSpeedTest()

		if err != nil {
			log.Fatalln(err)
		}

		j, err := json.Marshal(speedInfo)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(j))
	}()

	// NETWORK INFO
	wg.Add(1)
	go func() {
		defer wg.Done()
		networkInfo, err := CheckNetworkInfo()

		if err != nil {
			log.Fatalln(err)
		}

		j, err := json.Marshal(networkInfo)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(j))
	}()

	wg.Wait()
}
