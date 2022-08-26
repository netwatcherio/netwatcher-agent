package main

import (
	"errors"
	pj "github.com/hokaccha/go-prettyjson"
	"github.com/jackpal/gateway"
	"github.com/showwin/speedtest-go/speedtest"
	"github.com/tonobo/mtr/pkg/icmp"
	"github.com/tonobo/mtr/pkg/mtr"
	"log"
	"math"
	"math/rand"
	"net"
	"netwatcher-agent/models"
)

func CheckICMP(t models.IcmpTarget) {
	ipAddr := net.IPAddr{IP: net.ParseIP(t.Address)}

	seq := rand.Intn(math.MaxUint16)
	id := rand.Intn(math.MaxUint16) & 0xffff
	hop, _ := icmp.SendICMP(srcAddr, &ipAddr, t.Address, ttl, id, timeout, seq)
	t.Result.ElapsedMilliseconds = hop.Elapsed.Milliseconds()
}

func CheckMTR(t *models.MtrTarget, count int) {
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

func CheckNetworkInfo() (models.NetworkInfo, error) {
	user, err := speedtest.FetchUserInfo()
	if err != nil {
		return models.NetworkInfo{}, err
	}

	localInterface, err := gateway.DiscoverInterface()
	if err != nil {
		return models.NetworkInfo{}, err
	}

	return models.NetworkInfo{
		LocalSubnet:      localInterface.String(),
		PublicAddress:    user.IP,
		InternetProvider: user.Isp,
		Lat:              user.Lat,
		Long:             user.Lon,
	}, nil
}

func RunSpeedTest() (models.SpeedTestInfo, error) {
	user, err := speedtest.FetchUserInfo()
	if err != nil {
		return models.SpeedTestInfo{}, err
	}
	serverList, err := speedtest.FetchServers(user)
	if err != nil {
		return models.SpeedTestInfo{}, err
	}
	targets, err := serverList.FindServer([]int{})
	if err != nil {
		return models.SpeedTestInfo{}, err
	}

	for _, s := range targets {
		s.PingTest()
		s.DownloadTest(false)
		s.UploadTest(false)

		return models.SpeedTestInfo{
			Latency: s.Latency,
			DLSpeed: s.DLSpeed,
			ULSpeed: s.ULSpeed,
			Server:  s.Name,
			Host:    s.Host,
		}, nil
	}

	return models.SpeedTestInfo{}, errors.New("unable to reach Ookla")
}
