package main

type AgentConfig struct {
	PingTargets  []string `json:"ping_targets"`
	TraceTargets []string `json:"trace_targets"`
	PingInterval int64    //seconds

}
