package main

import (
	"github.com/sagostin/netwatcher-agent/agent_models"
	log "github.com/sirupsen/logrus"
	"github.com/tonobo/mtr/pkg/mtr"
	"sync"
	"time"
)

func TestMtrTargets(t []*agent_models.MtrTarget, triggered bool) {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for n := range t {
			res, err := CheckMTR(t[n], 5)
			if err != nil {
				log.Fatal(err)
			}

			t[n].Result.Mtr = mtr.MTR{
				SrcAddress: res.SrcAddress,
				Address:    res.Address,
				Statistic:  res.Statistic,
			}
			t[n].Result.Timestamp = time.Now()
			t[n].Result.Triggered = triggered
		}
	}()

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
