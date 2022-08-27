package main

import (
	"bytes"
	"encoding/json"
	"github.com/sagostin/netwatcher-agent/agent_models"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
)

/*
TODO

api that has get and post functions to allow sending and receiving commands/updates
poll configuration update every 5 minutes
push to servers every 2 minutes with data collected (configurable on frontend?)
front end/backend server will do the most work processing and sending alerts regarding sites and such

*/

func PushIcmp(t []*agent_models.IcmpTarget) (agent_models.ApiConfigResponse, error) {
	var apiUrl = os.Getenv("API_URL") + "/agent/update/icmp"

	j, err := json.Marshal(t)
	if err != nil {
		return agent_models.ApiConfigResponse{}, err
	}

	// TODO include authentication information
	resp, err := http.Post(apiUrl, "application/json",
		bytes.NewBuffer(j))
	if err != nil {
		return agent_models.ApiConfigResponse{}, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return agent_models.ApiConfigResponse{}, err
	}

	log.Warn(string(body))

	cfgResp := agent_models.ApiConfigResponse{}
	err = json.Unmarshal(body, &cfgResp)
	if err != nil {
		return agent_models.ApiConfigResponse{}, err
	}

	// TODO unmarshal body to api response model
	return cfgResp, nil
}
