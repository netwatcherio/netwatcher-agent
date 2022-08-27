package main

import (
	"github.com/joho/godotenv"
	"github.com/sagostin/netwatcher-agent/agent_models"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	CheckConfig *agent_models.CheckConfig
)

func main() {
	var err error
	if err != nil {
		log.Fatal(err)
	}

	log.SetFormatter(&log.TextFormatter{})

	godotenv.Load()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	signal.Notify(signals, syscall.SIGKILL)
	go func() {
		s := <-signals
		log.Fatal("Received Signal: %s", s)
		shutdown()
		os.Exit(1)
	}()

	var wg sync.WaitGroup
	log.Infof("Starting NetWatcher Agent...")

	StartScheduler()

	wg.Wait()
}

func shutdown() {
	log.Fatal("Shutting down NetWatcher Agent...")
}

/*
https://stackoverflow.com/questions/51717409/is-there-any-way-to-sign-the-windows-executables-generated-by-the-go-compiler How to Sign Windows Applications to prevent AV from detecting it...??
AVs don't like GoLang because the weird binary structure
*/

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
