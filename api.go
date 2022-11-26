package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/netwatcherio/netwatcher-agent/agent_models"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

// todo change to post and actually have authentication
func GetConfig() (*agent_models.AgentConfig, error) {
	// TODO include authentication information
	resp, err := http.Get(ApiUrl + "/v1/agent/config/" + os.Getenv("PIN") + "/" + os.Getenv("HASH"))
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Println(string(body))

	var apiCfg *agent_models.ApiConfigResponse
	err = json.Unmarshal(body, &apiCfg)
	if err != nil {
		return nil, err
	}

	if apiCfg.Response == 200 {
		return &apiCfg.Config, nil
	} else {
		return nil, errors.New("unable to get config")
	}

	// TODO unmarshal body to api response modes
}

func PostNetworkInfo(t *agent_models.NetworkInfo) (agent_models.ApiResponse, error) {
	verifyData := agent_models.ApiPushData{
		Pin:       os.Getenv("PIN"),
		Hash:      os.Getenv("HASH"),
		Data:      t,
		Timestamp: time.Now(),
	}

	j, err := json.Marshal(verifyData)
	if err != nil {
		return agent_models.ApiResponse{}, err
	}

	resp, err := postData(j, "/v1/agent/update/network")
	return resp, err
}

func PostSpeedTest(t *agent_models.SpeedTestInfo) (agent_models.ApiResponse, error) {
	verifyData := agent_models.ApiPushData{
		Pin:       os.Getenv("PIN"),
		Hash:      os.Getenv("HASH"),
		Data:      t,
		Timestamp: time.Now(),
	}

	j, err := json.Marshal(verifyData)
	if err != nil {
		return agent_models.ApiResponse{}, err
	}

	resp, err := postData(j, "/v1/agent/update/speedtest")
	return resp, err
}

func PostMtr(t []*agent_models.MtrTarget) (agent_models.ApiResponse, error) {
	verifyData := agent_models.ApiPushData{
		Pin:       os.Getenv("PIN"),
		Hash:      os.Getenv("HASH"),
		Data:      t,
		Timestamp: time.Now(),
	}

	j, err := json.Marshal(verifyData)
	log.Errorf("%s", j)

	resp, err := postData(j, "/v1/agent/update/mtr")
	return resp, err
}

func PostIcmp(t []agent_models.IcmpTarget) (agent_models.ApiResponse, error) {
	verifyData := agent_models.ApiPushData{
		Pin:       os.Getenv("PIN"),
		Hash:      os.Getenv("HASH"),
		Data:      t,
		Timestamp: time.Now(),
	}

	j, err := json.Marshal(verifyData)

	resp, err := postData(j, "/v1/agent/update/icmp")
	return resp, err
}

func postData(b []byte, apiPath string) (agent_models.ApiResponse, error) {
	// TODO include authentication information
	resp, err := http.Post(ApiUrl+apiPath, "application/json",
		bytes.NewBuffer(b))
	if err != nil {
		return agent_models.ApiResponse{}, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return agent_models.ApiResponse{}, err
	}

	//log.Warn(string(body))

	cfgResp := agent_models.ApiResponse{}
	err = json.Unmarshal(body, &cfgResp)
	if err != nil {
		return agent_models.ApiResponse{}, err
	}

	// TODO unmarshal body to api response model
	return cfgResp, nil
}
