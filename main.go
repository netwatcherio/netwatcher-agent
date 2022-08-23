package main

import (
	"fmt"
	_ "github.com/joho/godotenv"
	"sync"
)
import _ "log"

//import "os"

/*

TODO

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

	var t = []mtrTarget{
		{
			Address: "1.1.1.1",
		},
	}

	for _, st := range t {
		wg.Add(1)
		go func() {
			defer wg.Done()
			CheckMTR(&st)
		}()
		wg.Wait()

		fmt.Printf("Hop info %s %s", st.Address, st.Result)
	}

	var t2 = []icmpTarget{
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

}
