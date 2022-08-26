package agent_models

// eventally add more types of targets checks (SNMP, HTTP, etc.)
// nmap???

type TargetConfig struct {
	PingTargets  []string `json:"ping_targets"`
	TraceTargets []string `json:"trace_targets"`
	PingInterval int64    `json:"ping_interval"` // seconds
}

// send pin to sever, if hash doesn't exist in server or is blank, have
// server generate a hash as response in json, or with error,
// if server returns hash and no error, uninstall program, or regenerate hash??
/*
{pin: "123456789", hash: "", }
*/

type AgentVerify struct {
	Pin  string `json:pin`
	Hash string `json:hash`
}
