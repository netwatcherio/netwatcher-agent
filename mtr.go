package main

import (
	"github.com/netwatcherio/netwatcher-agent/agent_models"
	log "github.com/sirupsen/logrus"
	"github.com/tonobo/mtr/pkg/mtr"
	"sync"
	"time"
)

func TestMtrTargets(t []*agent_models.MtrTarget, triggered bool) {
	var wg sync.WaitGroup

	for _, tn := range t {
		wg.Add(1)
		go func(tn1 *agent_models.MtrTarget) {
			defer wg.Done()
			res, err := CheckMTR(tn1, 5)
			if err != nil {
				log.Fatal(err)
			}

			tn1.Result.Mtr = mtr.MTR{
				SrcAddress: res.SrcAddress,
				Address:    res.Address,
				Statistic:  res.Statistic,
			}
			tn1.Result.StopTimestamp = time.Now()
			tn1.Result.Triggered = triggered
		}(tn)
	}

	wg.Wait()
}

func CheckMTR(t *agent_models.MtrTarget, count int) (*mtr.MTR, error) {
	m, ch, err := mtr.NewMTR(t.Address, srcAddr, timeout, interval, hopSleep,
		maxHops, maxUnknownHops, ringBufferSize, ptrLookup)
	if err != nil {
		return nil, err
	}

	go func(ch chan struct{}) {
		for {
			<-ch
		}
	}(ch)
	m.Run(ch, count)

	return m, nil
}
