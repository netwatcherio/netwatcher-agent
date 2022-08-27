package agent_models

import (
	"github.com/tonobo/mtr/pkg/mtr"
	"time"
)

// NetworkInfo network info such as subnet, local network, public ip,
// and isp, and lat and long
type NetworkInfo struct {
	LocalSubnet      string    `json:"local_subnet"`
	PublicAddress    string    `json:"public_address"`
	DefaultGateway   string    `json:"default_gateway"`
	InternetProvider string    `json:"internet_provider"`
	Lat              string    `json:"lat"`
	Long             string    `json:"long"`
	Timestamp        time.Time `json:"timestamp"`
}

// SpeedTestInfo
//TODO log how long it took and then timestamp of when it was started and finished
type SpeedTestInfo struct {
	Latency   time.Duration `json:"latency"`
	DLSpeed   float64       `json:"dl_speed"`
	ULSpeed   float64       `json:"ul_speed"`
	Server    string        `json:"server"`
	Host      string        `json:"host"`
	Timestamp time.Time     `json:"timestamp"`
}

type MtrTarget struct {
	Address string `json:"address"`
	Result  struct {
		Triggered bool      `json:"triggered"`
		Mtr       mtr.MTR   `json:"mtr"`
		Timestamp time.Time `json:"timestamp"`
	} `json:"result"`
}

type IcmpTarget struct {
	Address string `json:"address"`
	Result  struct {
		Timestamp time.Time  `json:"timestamp"`
		Data      []IcmpData `json:"data"`
		Metrics   struct {
			Average     time.Duration `json:"average"`
			Max         time.Duration `json:"max"`
			Min         time.Duration `json:"min"`
			LossPercent int           `json:"loss_percent"`
		} `json:"metrics"`
	} `json:"result"`
}

type IcmpData struct {
	Elapsed   time.Duration `json:"elapsed"`
	Success   bool          `json:"success"`
	Timestamp time.Time     `json:"timestamp"`
}
