package agent_models

import (
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
