package agent_models

import "time"

type ApiResponse struct {
	Response int    `json:"response"`
	Error    string `json:"error"`
}

type ApiConfigResponse struct {
	// todo add deactivation parameter
	// make it auto uninstall

	Response  int         `json:"response"`
	Error     string      `json:"error"`
	Config    AgentConfig `json:"data"`
	NewAgent  bool        `json:"new_agent"`
	AgentHash string      `json:"agent_hash"`
}

type ApiPushData struct {
	Pin       string      `json:"pin"`
	Hash      string      `json:"hash"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:data`
}
