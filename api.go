package main

import (
	"bytes"
	"encoding/json"
	"github.com/sagostin/netwatcher-agent/agent_models"
	"io/ioutil"
	"net/http"
)

/*
TODO
api that has get and post functions to allow sending and receiving commands/updates
poll configuration update every 5 minutes
push to servers every 2 minutes with data collected (configurable on frontend?)
front end/backend server will do the most work processing and sending alerts regarding sites and such
*/

func PostNetworkInfo(t *agent_models.NetworkInfo) (agent_models.ApiConfigResponse, error) {
	j, err := json.Marshal(t)
	if err != nil {
		return agent_models.ApiConfigResponse{}, err
	}

	resp, err := postData(j, "/v1/agent/update/network")
	return resp, err
}

func PostSpeedTest(t *agent_models.SpeedTestInfo) (agent_models.ApiConfigResponse, error) {
	j, err := json.Marshal(t)
	if err != nil {
		return agent_models.ApiConfigResponse{}, err
	}

	resp, err := postData(j, "/v1/agent/update/speedtest")
	return resp, err
}

func PostMtr(t []*agent_models.MtrTarget) (agent_models.ApiConfigResponse, error) {
	j, err := json.Marshal(t)
	if err != nil {
		return agent_models.ApiConfigResponse{}, err
	}

	resp, err := postData(j, "/v1/agent/update/mtr")
	return resp, err
}

func PostIcmp(t []*agent_models.IcmpTarget) (agent_models.ApiConfigResponse, error) {
	j, err := json.Marshal(t)
	if err != nil {
		return agent_models.ApiConfigResponse{}, err
	}

	resp, err := postData(j, "/v1/agent/update/icmp")
	return resp, err
}

func postData(b []byte, apiPath string) (agent_models.ApiConfigResponse, error) {
	// TODO include authentication information
	resp, err := http.Post(ApiUrl+apiPath, "application/json",
		bytes.NewBuffer(b))
	if err != nil {
		return agent_models.ApiConfigResponse{}, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return agent_models.ApiConfigResponse{}, err
	}

	//log.Warn(string(body))

	cfgResp := agent_models.ApiConfigResponse{}
	err = json.Unmarshal(body, &cfgResp)
	if err != nil {
		return agent_models.ApiConfigResponse{}, err
	}

	// TODO unmarshal body to api response model
	return cfgResp, nil
}
