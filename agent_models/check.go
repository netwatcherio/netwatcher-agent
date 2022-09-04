package agent_models

import (
	"github.com/tonobo/mtr/pkg/mtr"
	"time"
)

// NetworkInfo network info such as subnet, local network, public ip,
// and isp, and lat and long
type NetworkInfo struct {
	LocalAddress     string    `json:"local_address"bson:"local_address"`
	DefaultGateway   string    `json:"default_gateway"bson:"default_gateway"`
	PublicAddress    string    `json:"public_address"bson:"public_address"`
	InternetProvider string    `json:"internet_provider"bson:"internet_provider"`
	Lat              string    `json:"lat"bson:"lat"`
	Long             string    `json:"long"bson:"long"`
	Timestamp        time.Time `json:"timestamp"bson:"timestamp"`
}

// SpeedTestInfo
//TODO log how long it took and then timestamp of when it was started and finished
type SpeedTestInfo struct {
	Latency   time.Duration `json:"latency"bson:"latency"`
	DLSpeed   float64       `json:"dl_speed"bson:"dl_speed"`
	ULSpeed   float64       `json:"ul_speed"bson:"ul_speed"`
	Server    string        `json:"server"bson:"server"`
	Host      string        `json:"host"bson:"host"`
	Timestamp time.Time     `json:"timestamp"bson:"timestamp"`
}

type MtrTarget struct {
	Address string `json:"address"bson:"address"`
	Result  struct {
		Triggered      bool      `json:"triggered"bson:"triggered"`
		Mtr            mtr.MTR   `json:"mtr"bson:"mtr"`
		StartTimestamp time.Time `json:"start_timestamp"bson:"start_timestamp"`
		StopTimestamp  time.Time `json:"stop_timestamp"bson:"stop_timestamp"`
	} `json:"result"bson:"result"`
}

type IcmpTarget struct {
	Address string `json:"address"bson:"address"`
	Result  struct {
		StartTimestamp time.Time  `json:"start_timestamp"bson:"start_timestamp"`
		StopTimestamp  time.Time  `json:"stop_timestamp"bson:"stop_timestamp"`
		Data           []IcmpData `json:"data"bson:"data"`
		Metrics        struct {
			LatencyAverage time.Duration `json:"latency_average"bson:"latency_average"`
			LatencyMax     time.Duration `json:"latency_max"bson:"latency_max"`
			LatencyMin     time.Duration `json:"latency_min"bson:"latency_min"`
			LossPercent    int           `json:"loss_percent"bson:"loss_percent"`
			JitterAverage  time.Duration `json:"jitter_average"bson:"jitter_average"`
			JitterMax      time.Duration `json:"jitter_max"bson:"jitter_max"`
		} `json:"metrics"bson:"metrics"`
	} `json:"result"bson:"result"`
}

type IcmpData struct {
	Elapsed   time.Duration `json:"elapsed"bson:"elapsed"'`
	Success   bool          `json:"success"bson:"success"`
	Timestamp time.Time     `json:"timestamp"bson:"timestamp"`
}
