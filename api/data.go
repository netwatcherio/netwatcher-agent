package api

import "github.com/netwatcherio/netwatcher-agent/checks"

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
	// the agent will grab the latest configuration when first initializing

	// check if pin isn't empty and id is, and send get request to get the id to initialize the agent
	if (a.PIN != "" && a.ID == "") || (a.PIN != "" && a.ID != "") {
		// send get request only assuming the data only contains the id
		// the server WILL respond with the same sort of data except with the hash, or an error if it fails
		// then update the env file and id in the struct object

		// the client will send "none" as the password when first initializing
		err := a.Client.Request("GET", "/api/v2/agent/init", a, a)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *Data) Push(data []checks.CheckData) error {
	a.Checks = data

	err := a.Client.Request("POST", "/api/v2/agent/push", a, a)
	if err != nil {
		return err
	}

	return nil
}
