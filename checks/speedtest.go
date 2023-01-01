package checks

import (
	"errors"
	"github.com/showwin/speedtest-go/speedtest"
	"time"
)

type SpeedTest struct {
	Latency   time.Duration `json:"latency"bson:"latency"`
	DLSpeed   float64       `json:"dl_speed"bson:"dl_speed"`
	ULSpeed   float64       `json:"ul_speed"bson:"ul_speed"`
	Server    string        `json:"server"bson:"server"`
	Host      string        `json:"host"bson:"host"`
	Timestamp time.Time     `json:"timestamp"bson:"timestamp"`
}

func (s1 *SpeedTest) Check() error {
	user, err := speedtest.FetchUserInfo()
	if err != nil {
		return err
	}
	serverList, err := speedtest.FetchServers(user)
	if err != nil {
		return err
	}
	targets, err := serverList.FindServer([]int{})
	if err != nil {
		return err
	}

	for _, s := range targets {
		s.PingTest()
		s.DownloadTest(false)
		s.UploadTest(false)

		s1.Latency = s.Latency
		s1.DLSpeed = s.DLSpeed
		s1.ULSpeed = s.ULSpeed
		s1.Server = s.Name
		s1.Host = s.Host
		s1.Timestamp = time.Now()

		return nil
	}

	return errors.New("unable to reach Ookla")
}
