package main

import (
	"github.com/netwatcherio/netwatcher-agent/agent_models"
	log "github.com/sirupsen/logrus"
	"github.com/tonobo/mtr/pkg/icmp"
	"math"
	"math/rand"
	"net"
	"sync"
	"time"
)

// TODO DSCP Tags? https://github.com/rclone/rclone/issues/755

func CheckICMP(t *agent_models.IcmpTarget) (agent_models.IcmpData, error) {
	ipAddr := net.IPAddr{IP: net.ParseIP(t.Address)}

	seq := rand.Intn(math.MaxUint16)
	id := rand.Intn(math.MaxUint16) & 0xffff
	hop, err := icmp.SendICMP(srcAddr, &ipAddr, t.Address, ttl, id, timeout, seq)
	if err != nil {
		return agent_models.IcmpData{
			Success:   false,
			Timestamp: time.Now(),
		}, err
	}

	icmpData := agent_models.IcmpData{
		Elapsed:   hop.Elapsed,
		Success:   hop.Success,
		Timestamp: time.Now(),
	}

	return icmpData, nil
}

func TestIcmpTargets(t []*agent_models.IcmpTarget, count int, interval int) {
	var wg sync.WaitGroup

	for n := range t {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < count; i++ {
				icmp, err := CheckICMP(t[n])
				if err != nil {
					//  read ip 0.0.0.0: raw-read ip4 0.0.0.0: i/o timeout
					log.Errorf("%s", err)
				}

				t[n].Result.Data = append(t[n].Result.Data, icmp)
				t[n].Result.StopTimestamp = time.Now()
				time.Sleep(time.Duration(int(time.Second) * interval))
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
		wg.Add(5)
		// Latency Average
		go func() {
			defer wg.Done()
			var average = 0
			// TODO take into account the packet loss (if any?)
			for _, m := range t[n].Result.Data {
				average = average + int(m.Elapsed)
			}
			if len(t[n].Result.Data) > 0 {
				average = average / len(t[n].Result.Data)
				t[n].Result.Metrics.LatencyAverage = time.Duration(average)
			}
		}()
		// Latency Maximum
		go func() {
			defer wg.Done()
			var max = 0
			for _, m := range t[n].Result.Data {
				if max < int(m.Elapsed) {
					max = int(m.Elapsed)
				}
			}
			t[n].Result.Metrics.LatencyMax = time.Duration(max)
		}()
		// Latency Minimum
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
			t[n].Result.Metrics.LatencyMin = time.Duration(min)
		}()
		// Packet Loss Percentage
		go func() {
			defer wg.Done()
			var lossPercent = 0
			for _, m := range t[n].Result.Data {
				if !m.Success {
					lossPercent++
				}
			}
			if len(t[n].Result.Data) > 0 {
				lossPercent = lossPercent / len(t[n].Result.Data)
				t[n].Result.Metrics.LossPercent = lossPercent
			}
		}()
		// Jitter Average
		go func() {
			defer wg.Done()
			var jitterAvg = 0
			var prev = 0
			var jitterC = 0
			//var jitterVals []int
			for _, m := range t[n].Result.Data {
				if m.Success {
					if prev == 0 {
						prev = int(m.Elapsed)
					} else {
						if prev > int(m.Elapsed) {
							jitterAvg = jitterAvg + (prev - int(m.Elapsed))
							jitterC = jitterC + 1
							//jitterVals = append(jitterVals, jitterValVPrev)
						} else if int(m.Elapsed) > prev {
							jitterAvg = jitterAvg + (int(m.Elapsed) - prev)
							jitterC = jitterC + 1
							//jitterVals = append(jitterVals, jitterValVPrev)
						}
						prev = int(m.Elapsed)
					}
				}
			}

			if jitterC > 0 && jitterAvg > 0 {
				jitterAvg = jitterAvg / jitterC
			}
			t[n].Result.Metrics.JitterAverage = time.Duration(jitterAvg)
		}()
		// TODO jitter max, and jitter 95 percentile
	}

	wg.Wait()
}
