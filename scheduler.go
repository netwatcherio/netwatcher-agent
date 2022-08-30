package main

import (
	"encoding/json"
	"fmt"
	"github.com/sagostin/netwatcher-agent/agent_models"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

func StartScheduler() {
	var wg sync.WaitGroup
	/*
	 1. update config x minutes and first start using apikey
	 2. make go routines to run mtr checks
	 3. make go routines (every 5 seconds?) to check icmp
	*/

	// rewrite to pull using api get request
	CheckConfig := agent_models.CheckConfig{
		PingTargets:      nil,
		TraceTargets:     nil,
		PingInterval:     2,
		SpeedTestPending: true,
		TraceInterval:    5, // minutes
	}

	CheckConfig.TraceTargets = append(CheckConfig.TraceTargets, "1.1.1.1")
	CheckConfig.PingTargets = append(CheckConfig.PingTargets, "1.1.1.1")

	wg.Add(1)
	go func() {
		defer wg.Done()
		runIcmpCheck(&CheckConfig, 2)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		runSpeedTestCheck(&CheckConfig)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		runMtrCheck(&CheckConfig)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		runNetworkQuery()
	}()

	//TODO add a heartbeat function to keep track of if the machine is online, etc.

	wg.Wait()
}

func runNetworkQuery() {
	var wg sync.WaitGroup

	var nInfo *agent_models.NetworkInfo

	for true {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Infof("Running Network Info query...")

			networkInfo, err := CheckNetworkInfo()
			if err != nil {
				log.Errorf("%s", err)
			}
			nInfo = networkInfo
		}()
		wg.Wait()
		// Upload to server, check if it fails or not,
		// then if it does, save to temporary list
		// for later upload
		resp, err := PostNetworkInfo(nInfo)
		if err != nil || resp.Response == 404 {
			// TODO save to queue
			log.Errorf("Failed to push Network Information information.")
		}

		if resp.Response == 200 {
			log.Infof("Pushed Network information.")
		}

		/*j, _ := json.Marshal(resp)
		log.Infof("%s", j)*/

		time.Sleep(time.Duration(int(time.Minute) * 30))
	}
}

func runMtrCheck(t *agent_models.CheckConfig) {
	var wg sync.WaitGroup

	for true {
		var mtrTargets []*agent_models.MtrTarget

		for n := range t.TraceTargets {
			mtrTargets = append(mtrTargets, &agent_models.MtrTarget{
				Address: t.TraceTargets[n],
			})
			mtrTargets[n].Result.StartTimestamp = time.Now()
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Infof("Running MTR check...")
			if t.TraceInterval < 5 {
				t.TraceInterval = 5
			}
			TestMtrTargets(mtrTargets, false)

			for _, st := range mtrTargets {
				_, err := json.Marshal(st)
				if err != nil {
					log.Fatal(err)
				}
				//fmt.Printf("%s\n", string(j))
			}
		}()
		wg.Wait()
		// Upload to server, check if it fails or not,
		// then if it does, save to temporary list
		// for later upload
		resp, err := PostMtr(mtrTargets)
		if err != nil || resp.Response == 404 {
			// TODO save to queue
			log.Errorf("Failed to push MTR information.")
		}

		if resp.Response == 200 {
			log.Infof("Pushed MTR information.")
		}

		/*j, _ := json.Marshal(resp)
		log.Infof("%s", j)*/

		time.Sleep(time.Duration(int(time.Minute) * t.TraceInterval))
	}
}

func runSpeedTestCheck(config *agent_models.CheckConfig) {
	var wg sync.WaitGroup

	for true {
		if config.SpeedTestPending {
			wg.Add(1)
			go func() {
				defer wg.Done()
				log.Infof("Running SpeedTest...")
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
			wg.Wait()
			// TODO then upload and verify it worked, otherwise save and process later

			config.SpeedTestPending = false
			// sleep
			time.Sleep(time.Duration(int(time.Second) * 300))
		}
	}
}

func runIcmpCheck(t *agent_models.CheckConfig, count int) {
	var wg sync.WaitGroup

	for true {
		var pingTargets []*agent_models.IcmpTarget

		for n := range t.PingTargets {
			pingTargets = append(pingTargets, &agent_models.IcmpTarget{
				Address: t.PingTargets[n],
			})

			pingTargets[n].Result.StartTimestamp = time.Now()
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Infof("Running ICMP check...")
			if t.PingInterval < 2 {
				t.PingInterval = 2
			}
			TestIcmpTargets(pingTargets, count, t.PingInterval)

			for _, st := range pingTargets {
				_, err := json.Marshal(st)
				if err != nil {
					log.Errorf("%s", err)
				}
				//fmt.Printf("%s\n", string(j))
			}
		}()
		wg.Wait()
		// Upload to server, check if it fails or not,
		// then if it does, save to temporary list
		// for later upload
		resp, err := PostIcmp(pingTargets)
		if err != nil || resp.Response == 404 {
			// TODO save to queue
			log.Errorf("Failed to push ICMP information.")
		}

		if resp.Response == 200 {
			log.Infof("Pushed ICMP information.")
		}

		/*j, _ := json.Marshal(resp)
		log.Infof("%s", j)*/
	}
}
