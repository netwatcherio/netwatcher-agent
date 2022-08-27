package main

import (
	"encoding/json"
	"github.com/sagostin/netwatcher-agent/agent_models"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

/*
TODO

api that has get and post functions to allow sending and receiving commands/updates
poll configuration update every 5 minutes
push to servers every 2 minutes with data collected (configurable on frontend?)
front end/backend server will do the most work processing and sending alerts regarding sites and such

*/

func PushIcmp(t []*agent_models.IcmpTarget) error {
	var apiURL = os.Getenv("API_URL") + "update/icmp"

	j, err := json.Marshal(t)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.PostForm(apiURL, url.Values{"data": {string(j)}, "id": {"this is a test"}})
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// TODO unmarshal body to api response model

	return err
}
