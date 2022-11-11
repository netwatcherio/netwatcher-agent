package agent_models

import "time"

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
