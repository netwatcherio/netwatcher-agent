package main

import (
	"fmt"
	"github.com/showwin/speedtest-go/speedtest"
)

// network info such as subnet, local network, public ip, and isp, and lat and long
type NetworkInfo struct {
}

type SpeedTestInfo struct {
	Latency string `json:"latency"`
	DLSpeed string `json:"dl_speed"`
	ULSpeed string
}

func RunSpeedTest() SpeedTestInfo {
	user, _ := speedtest.FetchUserInfo()

	serverList, _ := speedtest.FetchServers(user)
	targets, _ := serverList.FindServer([]int{})

	for _, s := range targets {
		s.PingTest()
		s.DownloadTest(false)
		s.UploadTest(false)
		fmt.Printf("Latency: %s, Download: %f, Upload: %f\n", s.Latency, s.DLSpeed, s.ULSpeed)
		return SpeedTestInfo{}
	}

	return nil
}
