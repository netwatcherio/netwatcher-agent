package main

import (
	"fmt"
	"github.com/netwatcherio/netwatcher-agent/probes"
	"github.com/netwatcherio/netwatcher-agent/workers"
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

	var probeGetCh = make(chan []probes.Probe)
	var probeDataCh = make(chan probes.ProbeData)
	workers.InitProbeWorker(probeGetCh, probeDataCh)

	wsH := &ws.WebSocketHandler{
		Host:       os.Getenv("HOST"),
		HostWS:     os.Getenv("HOST_WS"),
		Pin:        os.Getenv("PIN"),
		ID:         os.Getenv("ID"),
		ProbeGetCh: probeGetCh,
	}
	wsH.InitWS()
	workers.InitProbeDataWorker(wsH.GetConnection(), probeDataCh)

	// todo handle if on start it isn't able to pull information from backend??
	// eg. power goes out but network fails to come up?

	go func(ws *ws.WebSocketHandler) {
		for {
			time.Sleep(time.Minute * 1)
			log.Info("Getting probes again...")
			ws.GetConnection().Emit("probe_get", []byte("please"))
		}
	}(wsH)

	// todo input channel into wsH for inbound/outbound data to be handled
	// if a list of probes is received, send it to the channel for inbound probes and such
	// once receiving probes, have it cycle through, set the unique id for it, if a different one exists as the same ID,
	//update/remove it, n use the new settings
	select {}
}

func shutdown() {
	log.Fatalf("Currently %d threads", runtime.NumGoroutine())
	log.Fatal("Shutting down NetWatcher Agent...")
}
