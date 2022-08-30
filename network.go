package main

import (
	"errors"
	"github.com/jackpal/gateway"
	"github.com/sagostin/netwatcher-agent/agent_models"
	"github.com/showwin/speedtest-go/speedtest"
	"time"
)

func RunSpeedTest() (*agent_models.SpeedTestInfo, error) {
	user, err := speedtest.FetchUserInfo()
	if err != nil {
		return nil, err
	}
	serverList, err := speedtest.FetchServers(user)
	if err != nil {
		return nil, err
	}
	targets, err := serverList.FindServer([]int{})
	if err != nil {
		return nil, err
	}

	for _, s := range targets {
		s.PingTest()
		s.DownloadTest(false)
		s.UploadTest(false)

		return &agent_models.SpeedTestInfo{
			Latency:   s.Latency,
			DLSpeed:   s.DLSpeed,
			ULSpeed:   s.ULSpeed,
			Server:    s.Name,
			Host:      s.Host,
			Timestamp: time.Now(),
		}, nil
	}

	return nil, errors.New("unable to reach Ookla")
}

func CheckNetworkInfo() (*agent_models.NetworkInfo, error) {
	user, err := speedtest.FetchUserInfo()
	if err != nil {
		return nil, err
	}

	localInterface, err := gateway.DiscoverInterface()
	if err != nil {
		return nil, err
	}

	localGateway, err := gateway.DiscoverGateway()
	if err != nil {
		return nil, err
	}

	return &agent_models.NetworkInfo{
		LocalAddress:     localInterface.String(),
		PublicAddress:    user.IP,
		InternetProvider: user.Isp,
		Lat:              user.Lat,
		Long:             user.Lon,
		DefaultGateway:   localGateway.String(),
		Timestamp:        time.Now(),
	}, nil
}
