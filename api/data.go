package api

import (
	"errors"
	"github.com/netwatcherio/netwatcher-agent/checks"
)

/*

ID is the object ID in the database, when authenticating, it will send a get request with only the pin on initialization
the server will respond, with the hash or an error if the pin has already been setup

once initialized, the id is saved and will be used for sending and receiving configuration updated
*/

type Data struct {
	Client
	PIN    string             `json:"pin,omitempty"`
	ID     string             `json:"id,omitempty"`
	Checks []checks.CheckData `json:"checks,omitempty"`
	Error  string             `json:"error,omitempty"`
}

func (c Client) Data() Data {
	return Data{
		Client: c,
	}
}

func (a *Data) Initialize() error {
	if (a.PIN != "" && a.ID == "") || (a.PIN != "" && a.ID != "") {
		err := a.Client.Request("POST", "/api/v2/config/", &a, &a)
		if err != nil {
			return err
		}

		if a.Error != "" {
			return errors.New(a.Error)
		}
		return nil
	}
	return errors.New("failed to meet requirements for auth")
}

func (a *Data) Push() error {
	err := a.Client.Request("POST", "/api/v2/agent/push", &a, &a)
	if err != nil {
		return err
	}

	return nil
}
