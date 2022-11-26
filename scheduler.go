package main

import (
	"encoding/json"
	"github.com/netwatcherio/netwatcher-agent/agent_models"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type API interface {
	GetConfig() (*agent_models.AgentConfig, error)
}

func StartScheduler() {

	/*
	 1. update config x minutes and first start using apikey
	 2. make go routines to run mtr checks
	 3. make go routines (every 5 seconds?) to check icmp
	*/

	for {
		// A local reference to the agent config
		var conf *agent_models.AgentConfig
		// Attempt to pull the agent config
		conf, err := GetConfig()
		// If an error occurs, a message is logged to console and the loop repeats after one minute
		if err != nil {
			log.WithError(err).Warnf("Unable to fetch configuration, trying again in 1 minutes")
			time.Sleep(time.Minute)
			continue
		}

		runChecks(*conf)

		time.Sleep(time.Minute * 5)
	}

	// TODO add a heartbeat function to keep track of if the machine is online, etc.

}

func runChecks(agentConfig agent_models.AgentConfig) {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		runIcmpCheck(&agentConfig)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runSpeedTestCheck(&agentConfig)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runMtrCheck(&agentConfig)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runNetworkQuery()
	}()

	wg.Wait()
}

func runNetworkQuery() {
	// var wg sync.WaitGroup
	//
	// for {
	// 	wg.Add(1)
	//
	// 	go func() {
	// 		defer wg.Done()
	log.Infof("Running Network Info query...")

	networkInfo, err := CheckNetworkInfo()
	if err != nil {
		log.Errorf("%s", err)
	} else {
		resp, err := PostNetworkInfo(networkInfo)
		if err != nil || resp.Response != 200 {
			// TODO save to queue
			log.Errorf("Failed to push Network Information information.")
		}

		if resp.Response == 200 {
			log.Infof("Pushed Network information.")
		}
	}
	// 	}()
	//
	// 	wg.Wait()
	// 	// Upload to server, check if it fails or not,
	// 	// then if it does, save to temporary list
	// 	// for later upload
	//
	// 	/*j, _ := json.Marshal(resp)
	// 	log.Infof("%s", j)*/
	//
	// 	time.Sleep(time.Duration(int(time.Minute) * 30))
	// }
}

func runMtrCheck(t *agent_models.AgentConfig) {
	for {
		log.Infof("Running MTR check...")
		if t.TraceInterval < 5 {
			t.TraceInterval = 5
		}
		c, err := TestMtrTargets(t.TraceTargets, false)
		if err != nil {
			log.Error(err)
		}

		resp, err := PostMtr(c)
		if err != nil || resp.Response != 200 {
			// TODO save to queue
			log.Errorf("Failed to push MTR information.")
		}

		if resp.Response == 200 {
			log.Infof("Pushed MTR information.")
		}

		j, _ := json.Marshal(resp)
		log.Infof("%s", j)
		time.Sleep(time.Duration(t.TraceInterval * int(time.Minute)))
	}
}

func runSpeedTestCheck(config *agent_models.AgentConfig) {
	// var wg sync.WaitGroup
	//
	// for {
	if config.SpeedTestPending {
		// wg.Add(1)
		// go func() {
		// 	defer wg.Done()
		log.Infof("Running SpeedTest...")
		speedInfo, err := RunSpeedTest()
		if err != nil {
			log.Fatalln(err)
		}
		// TODO verify it was sent other then save to queue if not sent
		// Upload to server, check if it fails or not,
		// then if it does, save to temporary list
		// for later upload
		resp, err := PostSpeedTest(speedInfo)
		if err != nil || resp.Response != 200 {
			// TODO save to queue
			log.Errorf("Failed to push speedtest information.")
		}

		if resp.Response == 200 {
			log.Infof("Pushed speedtest information.")
		}
		// }()
		// wg.Wait()
		config.SpeedTestPending = false
		// sleep
	}
	// 	time.Sleep(time.Duration(int(time.Second) * 300))
	// }
}

func runIcmpCheck(t *agent_models.AgentConfig) {
	for {
		log.Infof("Running ICMP check...")
		if t.PingInterval < 1 {
			t.PingInterval = 1
		}
		t2, err := TestIcmpTargets(t.PingTargets, t.PingInterval)
		if err != nil {
			log.Error(err)
		}

		// Upload to server, check if it fails or not,
		// then if it does, save to temporary list
		// for later upload
		resp, err := PostIcmp(t2)
		if err != nil || resp.Response != 200 {
			// TODO save to queue
			log.Errorf("Failed to push ICMP information.")
		}

		if resp.Response == 200 {
			log.Infof("Pushed ICMP information.")
		}
	}
}
