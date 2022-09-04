package agent_models

import "time"

type ApiResponse struct {
	Response int    `json:"response"bson:"response"`
	Error    string `json:"error"bson:"error"`
}

type ApiConfigResponse struct {
	// todo add deactivation parameter
	// make it auto uninstall

	Response  int         `json:"response"bson:"response"`
	Error     string      `json:"error"bson:"error"`
	Config    AgentConfig `json:"data"bson:"config"`
	NewAgent  bool        `json:"new_agent"bson:"new_agent"`
	AgentHash string      `json:"agent_hash"bson:"agent_hash"`
}

type ApiPushData struct {
	Pin       string      `json:"pin"bson:"pin"`
	Hash      string      `json:"hash"bson:"hash"`
	Timestamp time.Time   `json:"timestamp"bson:"timestamp"`
	Data      interface{} `json:"data"bson:"data"`
}
