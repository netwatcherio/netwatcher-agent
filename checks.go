package main

import (
	"errors"
	pj "github.com/hokaccha/go-prettyjson"
	"github.com/jackpal/gateway"
	"github.com/sagostin/netwatcher-agent/agent_models"
	"github.com/showwin/speedtest-go/speedtest"
	"github.com/tonobo/mtr/pkg/icmp"
	"github.com/tonobo/mtr/pkg/mtr"
	"log"
	"math"
	"math/rand"
	"net"
)

func CheckICMP(t *agent_models.IcmpTarget) {
	ipAddr := net.IPAddr{IP: net.ParseIP(t.Address)}

	seq := rand.Intn(math.MaxUint16)
	id := rand.Intn(math.MaxUint16) & 0xffff
	hop, _ := icmp.SendICMP(srcAddr, &ipAddr, t.Address, ttl, id, timeout, seq)
	t.Result.ElapsedMilliseconds = hop.Elapsed.Milliseconds()
}

func CheckMTR(t *agent_models.MtrTarget, count int) {
	m, ch, err := mtr.NewMTR(t.Address, srcAddr, timeout, interval, hopSleep,
		maxHops, maxUnknownHops, ringBufferSize, ptrLookup)
	if err != nil {
		log.Fatal(err)
	}

	go func(ch chan struct{}) {
		for {
			<-ch
		}
	}(ch)
	m.Run(ch, count)
	s, err := pj.Marshal(m)
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println(string(s))
	t.Result = string(s)
}

func CheckNetworkInfo() (agent_models.NetworkInfo, error) {
	user, err := speedtest.FetchUserInfo()
	if err != nil {
		return agent_models.NetworkInfo{}, err
	}

	localInterface, err := gateway.DiscoverInterface()
	if err != nil {
		return agent_models.NetworkInfo{}, err
	}

	return agent_models.NetworkInfo{
		LocalSubnet:      localInterface.String(),
		PublicAddress:    user.IP,
		InternetProvider: user.Isp,
		Lat:              user.Lat,
		Long:             user.Lon,
	}, nil
}

func RunSpeedTest() (agent_models.SpeedTestInfo, error) {
	user, err := speedtest.FetchUserInfo()
	if err != nil {
		return agent_models.SpeedTestInfo{}, err
	}
	serverList, err := speedtest.FetchServers(user)
	if err != nil {
		return agent_models.SpeedTestInfo{}, err
	}
	targets, err := serverList.FindServer([]int{})
	if err != nil {
		return agent_models.SpeedTestInfo{}, err
	}

	for _, s := range targets {
		s.PingTest()
		s.DownloadTest(false)
		s.UploadTest(false)

		return agent_models.SpeedTestInfo{
			Latency: s.Latency,
			DLSpeed: s.DLSpeed,
			ULSpeed: s.ULSpeed,
			Server:  s.Name,
			Host:    s.Host,
		}, nil
	}

	return agent_models.SpeedTestInfo{}, errors.New("unable to reach Ookla")
}
