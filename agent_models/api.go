package agent_models

type ApiResponse struct {
	Response int    `json:"response"`
	Error    string `json:"error"`
}

type ApiConfigResponse struct {
	Response  int         `json:"response"`
	Error     string      `json:"error"`
	Config    AgentConfig `json:"data"`
	NewAgent  bool        `json:"new_agent"`
	AgentHash string      `json:"agent_hash"`
}
