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

type CheckData struct {
	Type      string             `json:"type"bson:"type""`
	Target    string             `json:"address,omitempty"bson:"target,omitempty"`
	ID        primitive.ObjectID `json:"id"bson:"_id"`
	AgentID   primitive.ObjectID `json:"agent_id"bson:"agent_id"`
	Duration  int                `json:"interval,omitempty'"bson:"duration"`
	Count     int                `json:"count,omitempty"`
	Triggered bool               `json:"triggered"bson:"triggered,omitempty"`
	ToRemove  bool               `json:"to_remove"bson:"to_remove,omitempty"`
	Result    interface{}        `json:"result"bson:"result,omitempty"`
	Server    bool               `json:"server,omitempty"bson:"server,omitempty"`
}
