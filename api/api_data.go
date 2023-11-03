package api

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*

ID is the object ID in the database, when authenticating, it will send a get request with only the pin on initialization
the server will respond, with the hash or an error if the pin has already been setup

once initialized, the id is saved and will be used for sending and receiving configuration updated
*/

type CheckType string

const (
	CtRperf     CheckType = "RPERF"
	CtMtr       CheckType = "MTR"
	CtSpeedtest CheckType = "SPEEDTEST"
	CtNetinfo   CheckType = "NETINFO"
	CtPing      CheckType = "PING"
)

type CheckData struct {
	Target    string             `json:"target,omitempty"bson:"target,omitempty"`
	CheckID   primitive.ObjectID `json:"check"bson:"check"`
	AgentID   primitive.ObjectID `json:"agent"bson:"agent"`
	Triggered bool               `json:"triggered"bson:"triggered,omitempty"`
	Result    interface{}        `json:"result"bson:"result,omitempty"`
	Type      CheckType          `bson:"type"json:"type"`
}

type AgentCheck struct {
	Type      CheckType          `json:"type"bson:"type""`
	Target    string             `json:"target,omitempty"bson:"target,omitempty"`
	ID        primitive.ObjectID `json:"id,omitempty"bson:"_id"`
	AgentID   primitive.ObjectID `json:"agent"bson:"agent"`
	Duration  int                `json:"duration,omitempty'"bson:"duration,omitempty"`
	Count     int                `json:"count,omitempty"bson:"count,omitempty"`
	Triggered bool               `json:"triggered"bson:"triggered,omitempty"`
	Server    bool               `json:"server,omitempty"bson:"server,omitempty"`
	Pending   bool               `json:"pending,omitempty"bson:"pending,omitempty"`
	Interval  int                `json:"interval"bson:"interval"`
}

/*type Data struct {
	Client
}

func (c Client) Data() Data {
	return Data{
		Client: c,
	}
}

func (a *Data) Initialize(ar *ApiRequest) error {
	if (ar.PIN != "" && ar.ID == "") || (ar.PIN != "" && ar.ID != "") {
		err := a.Client.Request("POST", "/api/v2/config/", &ar, &ar)
		if err != nil {
			return err
		}

		if ar.Error != "" {
			return errors.New(ar.Error)
		}
		return nil
	}
	return errors.New("failed to meet requirements for auth")
}

func (a *Data) Push(ar *ApiRequest) error {
	err := a.Client.Request("POST", "/api/v2/agent/push", &ar, &ar)
	if err != nil {
		return err
	}

	return nil
}*/
