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

func (s1 *SpeedTest) Check(cd *CheckData) error {
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

	if len(targets) <= 0 {
		return errors.New("unable to reach Ookla")
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

	cd.Result = s1

	return nil
}
