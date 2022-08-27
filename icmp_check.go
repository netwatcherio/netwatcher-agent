package main

import (
	"github.com/sagostin/netwatcher-agent/agent_models"
	"log"
	"sync"
	"time"
)

func TestICMP(t []*agent_models.IcmpTarget, count int, interval int) {
	var wg sync.WaitGroup

	for n := range t {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < count; i++ {
				icmp, err := CheckICMP(t[n])
				if err != nil {
					log.Fatal(err)
				}

				t[n].Result.Data = append(t[n].Result.Data, icmp)

				/*j, err := json.Marshal(icmp)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(string(j))*/
				time.Sleep(time.Duration(interval))
			}
		}()
	}
	wg.Wait()
}

func CalculateMetrics(t []*agent_models.IcmpTarget) {
	var wg sync.WaitGroup

	for n := range t {
		wg.Add(1)
		for i := 0; i < 4; i++ {
			go func() {
				defer wg.Done()
				for i := 0; i < count; i++ {
					icmp, err := CheckICMP(t[n])
					if err != nil {
						log.Fatal(err)
					}

					t[n].Result.Data = append(t[n].Result.Data, icmp)

					/*j, err := json.Marshal(icmp)
					if err != nil {
						log.Fatal(err)
					}
					fmt.Println(string(j))*/
					time.Sleep(time.Duration(interval))
				}
			}()
		}
	}
	wg.Wait()

}
