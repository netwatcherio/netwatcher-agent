package checks

import "go.mongodb.org/mongo-driver/bson/primitive"

/*
- Data will be uploaded at set intervals, if it fails, it waits and retries in the next set interval, continuing to
add data to the queue.
- Individual checks are setup, such as ICMP to a target server, checking port uptimes, speedtests, mtr, twamp
- All checks are

Checks will have a type, target, result, interval, count

Mtr: target

*/

// CheckData used for processing current, ongoing and updated checks
type CheckData struct {
	Type      string             `json:"type"bson:"type""`
	Target    string             `json:"address,omitempty"bson:"target,omitempty"`
	ID        primitive.ObjectID `json:"id"bson:"_id"`
	AgentID   primitive.ObjectID `json:"agent_id"bson:"agent_id"`
	Duration  string             `json:"interval,omitempty'"bson:"duration"`
	Count     string             `json:"count,omitempty"`
	Triggered bool               `json:"triggered"bson:"triggered,omitempty"`
	ToRemove  bool               `json:"to_remove"bson:"to_remove,omitempty"`
	Pending   bool               `json:"pending"`                 // only used to see if a speedtest is waiting, maybe for other checks eventually
	Interval  string             `json:"interval"bson:"interval"` // in minutes, used for mtr checks and such
	Result    interface{}        `json:"result"bson:"result,omitempty"`
	Server    bool               `json:"server,omitempty"bson:"server,omitempty"`
}
