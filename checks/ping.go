package checks

import (
	"fmt"
	"github.com/netwatcherio/netwatcher-agent/api"
	probing "github.com/prometheus-community/pro-bing"
	"time"
)

type PingResult struct {
	// StartTime is the time that the check started at
	StartTimestamp time.Time `json:"start_timestamp"bson:"start_timestamp"`
	StopTimestamp  time.Time `json:"stop_timestamp"bson:"stop_timestamp"`
	// PacketsRecv is the number of packets received.
	PacketsRecv int `json:"packets_recv"bson:"packets_recv"`
	// PacketsSent is the number of packets sent.
	PacketsSent int `json:"packets_sent"bson:"packets_sent"`
	// PacketsRecvDuplicates is the number of duplicate responses there were to a sent packet.
	PacketsRecvDuplicates int `json:"packets_recv_duplicates"bson:"packets_recv_duplicates"`
	// PacketLoss is the percentage of packets lost.
	PacketLoss float64 `json:"packet_loss"bson:"packet_loss"`
	// Addr is the string address of the host being pinged.
	Addr string `json:"addr"bson:"addr"`
	// MinRtt is the minimum round-trip time sent via this pinger.
	MinRtt time.Duration `json:"min_rtt"bson:"min_rtt"`
	// MaxRtt is the maximum round-trip time sent via this pinger.
	MaxRtt time.Duration `json:"max_rtt"bson:"max_rtt"`
	// AvgRtt is the average round-trip time sent via this pinger.
	AvgRtt time.Duration `json:"avg_rtt"bson:"avg_rtt"`
	// StdDevRtt is the standard deviation of the round-trip times sent via
	// this pinger.
	StdDevRtt time.Duration `json:"std_dev_rtt"bson:"std_dev_rtt"`
}

func Ping(ac *api.AgentCheck, pingChan chan PingResult) {
	startTime := time.Now()

	pinger, err := probing.NewPinger(ac.Target)
	if err != nil {
		fmt.Println(err)
	}

	go func() {
		time.Sleep(60 * time.Second)
		pinger.Stop()
	}()

	pinger.SetPrivileged(true)

	pinger.OnFinish = func(stats *probing.Statistics) {
		fmt.Printf("\n--- %s ping statistics ---\n", stats.Addr)
		fmt.Printf("%d packets transmitted, %d packets received, %v%% packet loss\n",
			stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)
		fmt.Printf("round-trip min/avg/max/stddev = %v/%v/%v/%v\n",
			stats.MinRtt, stats.AvgRtt, stats.MaxRtt, stats.StdDevRtt)

		pingR := PingResult{
			StartTimestamp:        startTime,
			StopTimestamp:         time.Now(),
			PacketsRecv:           stats.PacketsRecv,
			PacketsSent:           stats.PacketsSent,
			PacketsRecvDuplicates: stats.PacketsRecvDuplicates,
			PacketLoss:            stats.PacketLoss,
			Addr:                  stats.Addr,
			MinRtt:                stats.MinRtt,
			MaxRtt:                stats.MaxRtt,
			AvgRtt:                stats.MinRtt,
			StdDevRtt:             stats.StdDevRtt,
		}

		pingChan <- pingR
	}

	err = pinger.Run()
	if err != nil {
		fmt.Println(err)
	}
}
