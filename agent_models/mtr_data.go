package agent_models

import (
	"time"
)

type MtrTarget struct {
	Address string `json:"address"bson:"address"`
	Result  struct {
		Triggered      bool           `json:"triggered"bson:"triggered"`
		Mtr            map[int]MtrHop `json:"mtr"bson:"mtr"`
		StartTimestamp time.Time      `json:"start_timestamp"bson:"start_timestamp"`
		StopTimestamp  time.Time      `json:"stop_timestamp"bson:"stop_timestamp"`
	} `json:"result"bson:"result"`
}

type MtrHop struct {
	Address  string `json:"address"bson:"address"`
	Sent     int    `json:"sent"bson:"sent"`
	Received int    `json:"received"bson:"received"`
	Last     string `bson:"last"json:"last"`
	Avg      string `bson:"avg"json:"avg"`
	Best     string `bson:"best"json:"best"`
	Worst    string `bson:"worst"json:"worst"`
}
