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
// TODO log how long it took and then timestamp of when it was started and finished
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
		StartTimestamp time.Time   `json:"start_timestamp"bson:"start_timestamp"`
		StopTimestamp  time.Time   `json:"stop_timestamp"bson:"stop_timestamp"`
		Metrics        IcmpMetrics `json:"metrics"bson:"metrics"`
	} `json:"result"bson:"result"`
}

type IcmpMetrics struct {
	Avg         string `json:"avg"bson:"avg"`
	Min         string `json:"min"bson:"min"`
	Max         string `json:"max"bson:"max"`
	Sent        int    `json:"sent"bson:"sent"`
	Received    int    `json:"received"bson:"received"`
	Loss        int    `json:"loss"bson:"loss"`
	Percent50   string `json:"percent_50"bson:"percent_50"`
	Percent90   string `json:"percent_90"bson:"percent_90"`
	Percent95   string `json:"percent_95"bson:"percent_95"`
	Percent99   string `json:"percent_99"bson:"percent_99"`
	Percent999  string `json:"percent_999"bson:"percent_999"`
	Percent9999 string `json:"percent_9999"bson:"percent_9999"`
}
