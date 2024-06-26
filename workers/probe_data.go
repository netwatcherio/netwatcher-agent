package workers

import (
	"encoding/json"
	"github.com/netwatcherio/netwatcher-agent/probes"
	"github.com/netwatcherio/netwatcher-agent/ws"
)

func InitProbeDataWorker(wsH *ws.WebSocketHandler, ch chan probes.ProbeData) {
	go func(cn *ws.WebSocketHandler, c chan probes.ProbeData) {
		for p := range ch {
			marshal, err := json.Marshal(p)
			if err != nil {
				return
			}
			wsH.GetConnection().Emit("probe_post", marshal)
		}
	}(wsH, ch)
}
