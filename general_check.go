package main

import (
	"errors"
	"github.com/jackpal/gateway"
	"github.com/sagostin/netwatcher-agent/agent_models"
	"github.com/showwin/speedtest-go/speedtest"
	"github.com/tonobo/mtr/pkg/icmp"
	"github.com/tonobo/mtr/pkg/mtr"
	"math"
	"math/rand"
	"net"
)

func CheckICMP(t *agent_models.IcmpTarget) (agent_models.IcmpData, error) {
	ipAddr := net.IPAddr{IP: net.ParseIP(t.Address)}

	seq := rand.Intn(math.MaxUint16)
	id := rand.Intn(math.MaxUint16) & 0xffff
	hop, err := icmp.SendICMP(srcAddr, &ipAddr, t.Address, ttl, id, timeout, seq)
	if err != nil {
		return agent_models.IcmpData{}, err
	}

	icmpData := agent_models.IcmpData{
		Elapsed: hop.Elapsed,
		Success: hop.Success,
	}

	return icmpData, nil
}

func CheckMTR(t *agent_models.MtrTarget, count int) (*mtr.MTR, error) {
	m, ch, err := mtr.NewMTR(t.Address, srcAddr, timeout, interval, hopSleep,
		maxHops, maxUnknownHops, ringBufferSize, ptrLookup)
	if err != nil {
		return nil, err
	}

	go func(ch chan struct{}) {
		for {
			<-ch
		}
	}(ch)
	m.Run(ch, count)

	return m, nil
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
		LocalSubnet:      localInterface.String(),
		PublicAddress:    user.IP,
		InternetProvider: user.Isp,
		Lat:              user.Lat,
		Long:             user.Lon,
		DefaultGateway:   localGateway.String(),
	}, nil
}

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
			Latency: s.Latency,
			DLSpeed: s.DLSpeed,
			ULSpeed: s.ULSpeed,
			Server:  s.Name,
			Host:    s.Host,
		}, nil
	}

	return nil, errors.New("unable to reach Ookla")
}
