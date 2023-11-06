package main

import (
	"fmt"
	"github.com/netwatcherio/netwatcher-agent/probes"
	"github.com/netwatcherio/netwatcher-agent/ws"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"runtime"
	"time"
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

	var probeGetCh = make(chan probes.Probe)
	go loadWorkers(probeGetCh)

	wsH := &ws.WebSocketHandler{
		Host:       os.Getenv("HOST"),
		HostWS:     os.Getenv("HOST_WS"),
		Pin:        os.Getenv("PIN"),
		ID:         os.Getenv("ID"),
		ProbeGetCh: probeGetCh,
	}
	wsH.InitWS()

	go func(ws *ws.WebSocketHandler) {
		for {
			log.Info("Getting again. 1")
			time.Sleep(10 * time.Second)
			log.Info("Getting again. 2")
			ws.GetConnection().Emit("probe_get", []byte("please"))
			time.Sleep(time.Minute * 5)
		}
	}(wsH)

	// todo input channel into wsH for inbound/outbound data to be handled
	// if a list of probes is received, send it to the channel for inbound probes and such
	// once receiving probes, have it cycle through, set the unique id for it, if a different one exists as the same ID,
	//update/remove it, n use the new settings
	select {}
}

func loadWorkers(pgCh <-chan probes.Probe) {
	for data := range pgCh {
		// Start a new goroutine for each piece of data received.
		go func(d probes.Probe) {
			log.Info("Loading probe: ", d.ID)
		}(data)
	}
}

func shutdown() {
	log.Fatalf("Currently %d threads", runtime.NumGoroutine())
	log.Fatal("Shutting down NetWatcher Agent...")
}
