package main

import (
	"github.com/sagostin/netwatcher-agent/agent_models"
	"log"
	"sync"
	"time"
)

func TestIcmpTargets(t []*agent_models.IcmpTarget, count int, interval int) {
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
				t[n].Result.Timestamp = time.Now()
				time.Sleep(time.Duration(interval))
			}
		}()
	}
	wg.Wait()
	wg.Add(1)
	go func() {
		defer wg.Done()
		calculateMetrics(t)
	}()
	wg.Wait()
}

func calculateMetrics(t []*agent_models.IcmpTarget) {
	var wg sync.WaitGroup

	for n := range t {
		wg.Add(4)
		go func() {
			defer wg.Done()
			var average = 0
			for _, m := range t[n].Result.Data {
				average = average + int(m.Elapsed)
			}
			average = average / len(t[n].Result.Data)
			t[n].Result.Metrics.Average = time.Duration(average)
		}()
		go func() {
			defer wg.Done()
			var max = 0
			for _, m := range t[n].Result.Data {
				if max < int(m.Elapsed) {
					max = int(m.Elapsed)
				}
			}
			t[n].Result.Metrics.Max = time.Duration(max)
		}()
		go func() {
			defer wg.Done()
			var min = 0
			for _, m := range t[n].Result.Data {
				if min == 0 {
					min = int(m.Elapsed)
				} else if min > int(m.Elapsed) {
					min = int(m.Elapsed)
				}
			}
			t[n].Result.Metrics.Min = time.Duration(min)
		}()
		go func() {
			defer wg.Done()
			var loss = 0
			for _, m := range t[n].Result.Data {
				if !m.Success {
					loss++
				}
			}
			t[n].Result.Metrics.Loss = loss
		}()
	}
	wg.Wait()

}
