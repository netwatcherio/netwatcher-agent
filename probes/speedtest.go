package probes

import (
	"errors"
	"github.com/netwatcherio/netwatcher-agent/api"
	"github.com/showwin/speedtest-go/speedtest"
	"time"
)

type SpeedTestResult struct {
	Latency   time.Duration `json:"latency"bson:"latency"`
	DLSpeed   float64       `json:"dl_speed"bson:"dl_speed"`
	ULSpeed   float64       `json:"ul_speed"bson:"ul_speed"`
	Server    string        `json:"server"bson:"server"`
	Host      string        `json:"host"bson:"host"`
	Timestamp time.Time     `json:"timestamp"bson:"timestamp"`
}

func SpeedTest(cd *api.AgentCheck) (SpeedTestResult, error) {
	var s1 SpeedTestResult
	user, err := speedtest.FetchUserInfo()
	if err != nil {
		return s1, err
	}
	serverList, err := speedtest.FetchServers(user)
	if err != nil {
		return s1, err
	}
	targets, err := serverList.FindServer([]int{})
	if err != nil {
		return s1, err
	}

	if len(targets) <= 0 {
		return s1, errors.New("unable to reach Ookla")
	}

	mainT := targets[0]

	mainT.PingTest()
	mainT.DownloadTest(false)
	mainT.UploadTest(false)

	s1.Latency = mainT.Latency
	s1.DLSpeed = mainT.DLSpeed
	s1.ULSpeed = mainT.ULSpeed
	s1.Server = mainT.Name
	s1.Host = mainT.Host
	s1.Timestamp = time.Now()

	return s1, nil
}
