package probes

import (
	"errors"
	"github.com/jackpal/gateway"
	"github.com/showwin/speedtest-go/speedtest"
	"time"
)

type NetworkInfoResult struct {
	LocalAddress     string    `json:"local_address"bson:"local_address"`
	DefaultGateway   string    `json:"default_gateway"bson:"default_gateway"`
	PublicAddress    string    `json:"public_address"bson:"public_address"`
	InternetProvider string    `json:"internet_provider"bson:"internet_provider"`
	Lat              string    `json:"lat"bson:"lat"`
	Long             string    `json:"long"bson:"long"`
	Timestamp        time.Time `json:"timestamp"bson:"timestamp"`
}

func NetworkInfo() (NetworkInfoResult, error) {
	var n NetworkInfoResult
	n.Timestamp = time.Now()

	user, err := speedtest.FetchUserInfo()
	if err != nil {
		return n, errors.New("unable to fetch general public network information")
	}

	n.PublicAddress = user.IP
	n.InternetProvider = user.Isp
	n.Lat = user.Lat
	n.Long = user.Lon

	defaultGateway, err := gateway.DiscoverGateway()
	if err != nil {
		return n, errors.New("could not discover local gateway address")
	}
	n.DefaultGateway = defaultGateway.String()

	localInterface, err := gateway.DiscoverInterface()
	if err != nil {
		return n, errors.New("could not discover local interface address")
	}
	n.LocalAddress = localInterface.String()

	return n, nil
}
