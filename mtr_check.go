package main

import (
	"github.com/sagostin/netwatcher-agent/agent_models"
	"github.com/tonobo/mtr/pkg/mtr"
	"log"
	"sync"
	"time"
)

func TestMtrTargets(t []*agent_models.MtrTarget) {
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
		}
	}()

	wg.Wait()
}
