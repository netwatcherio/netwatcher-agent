package main

import (
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
		wg.Wait()

		fmt.Printf("Hop info %s %s", st.Address, st.Result)
	}

	var t2 = []agent_models.IcmpTarget{
		{
			Address: "1.1.1.1",
		},
	}

	wg.Wait()

	for _, st := range t2 {
		go func() {
			defer wg.Done()
			CheckICMP(&st)
		}()
		wg.Wait()
		fmt.Printf("Time for %s - %vms", st.Address, st.Result.ElapsedMilliseconds)
	}

	speedInfo, err := RunSpeedTest()

	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(speedInfo)

}
