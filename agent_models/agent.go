package agent_models

// eventally add more types of targets checks (SNMP, HTTP, etc.)
// nmap???

type AgentConfig struct {
	PingTargets      []string `json:"ping_targets"bson:"ping_targets"`
	TraceTargets     []string `json:"trace_targets"bson:"trace_targets"`
	PingInterval     int      `json:"ping_interval"bson:"ping_interval"` // seconds
	SpeedTestPending bool     `json:"speedtest_pending"bson:"speedtest_pending"`
	ConfigPending    bool     `json:"config_pending"bson:"config_pending"`
	TraceInterval    int      `json:"trace_interval"bson:"trace_interval"` // minutes
}
