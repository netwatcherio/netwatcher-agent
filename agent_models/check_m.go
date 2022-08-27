package agent_models

import (
	"github.com/tonobo/mtr/pkg/mtr"
	"time"
)

// NetworkInfo network info such as subnet, local network, public ip,
// and isp, and lat and long
type NetworkInfo struct {
	LocalSubnet      string `json:"local_subnet"`
	PublicAddress    string `json:"public_address"`
	DefaultGateway   string `json:"default_gateway"`
	InternetProvider string `json:"internet_provider"`
	Lat              string `json:"lat"`
	Long             string `json:"long"`
}

// SpeedTestInfo
//TODO log how long it took and then timestamp of when it was started and finished
type SpeedTestInfo struct {
	Latency time.Duration `json:"latency"`
	DLSpeed float64       `json:"dl_speed"`
	ULSpeed float64       `json:"ul_speed"`
	Server  string        `json:"server"`
	Host    string        `json:"host"`
}

type MtrTarget struct {
	Address string `json:"address"`
	Result  struct {
		Mtr *mtr.MTR `json:"mtr"`
	} `json:"result"`
}

type IcmpTarget struct {
	Address string `json:"address"`
	Result  struct {
		Data    []IcmpData
		Metrics struct {
			Average time.Duration `json:"average"`
			Max     time.Duration `json:"max"`
			Min     time.Duration `json:"min"`
			Loss    int           `json:"loss"`
		}
	} `json:"result"`
}

type IcmpData struct {
	Elapsed   time.Duration `json:"elapsed"`
	Success   bool          `json:"success"`
	Timestamp time.Time     `json:"timestamp"`
}
