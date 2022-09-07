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

	log.Infof("len %v", len(t))
	for _, tn := range t {
		log.Infof("starting icmp for %s", tn.Address)
		wg.Add(1)
		go func(tn1 *agent_models.IcmpTarget) {
			defer wg.Done()
			for i := 0; i < count; i++ {
				icmp, err := CheckICMP(tn1)
				if err != nil {
					//  read ip 0.0.0.0: raw-read ip4 0.0.0.0: i/o timeout
					log.Errorf("%s", err)
				}
				tn1.Result.Data = append(tn1.Result.Data, icmp)
				tn1.Result.StopTimestamp = time.Now()
				time.Sleep(time.Duration(int(time.Second) * interval))
			}
			log.Infof("ending icmp for %s", tn1.Address)
		}(tn)
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
		go func(tn *agent_models.IcmpTarget) {
			defer wg.Done()
			var average = 0
			// TODO take into account the packet loss (if any?)
			for _, m := range tn.Result.Data {
				average = average + int(m.Elapsed)
			}
			if len(tn.Result.Data) > 0 {
				average = average / len(tn.Result.Data)
				tn.Result.Metrics.LatencyAverage = time.Duration(average)
			}
		}(t[n])
		// Latency Maximum
		go func(tn *agent_models.IcmpTarget) {
			defer wg.Done()
			var max = 0
			for _, m := range tn.Result.Data {
				if max < int(m.Elapsed) {
					max = int(m.Elapsed)
				}
			}
			tn.Result.Metrics.LatencyMax = time.Duration(max)
		}(t[n])
		// Latency Minimum
		go func(tn *agent_models.IcmpTarget) {
			defer wg.Done()
			var min = 0
			for _, m := range tn.Result.Data {
				if min == 0 {
					min = int(m.Elapsed)
				} else if min > int(m.Elapsed) {
					min = int(m.Elapsed)
				}
			}
			tn.Result.Metrics.LatencyMin = time.Duration(min)
		}(t[n])
		// Packet Loss Percentage
		go func(tn *agent_models.IcmpTarget) {
			defer wg.Done()
			var lossPercent = 0
			for _, m := range tn.Result.Data {
				if !m.Success {
					lossPercent++
				}
			}
			if len(tn.Result.Data) > 0 {
				lossPercent = lossPercent / len(tn.Result.Data)
				tn.Result.Metrics.LossPercent = lossPercent
			}
		}(t[n])
		// Jitter Average
		go func(tn *agent_models.IcmpTarget) {
			defer wg.Done()
			var jitterAvg = 0
			var prev = 0
			var jitterC = 0
			//var jitterVals []int
			for _, m := range tn.Result.Data {
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
			tn.Result.Metrics.JitterAverage = time.Duration(jitterAvg)
		}(t[n])
		// TODO jitter max, and jitter 95 percentile
	}
	wg.Wait()
}
