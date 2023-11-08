package workers

import (
	"encoding/json"
	"github.com/kataras/neffos"
	"github.com/netwatcherio/netwatcher-agent/probes"
)

func InitProbeDataWorker(conn *neffos.NSConn, ch chan probes.ProbeData) {
	go func(cn *neffos.NSConn, c chan probes.ProbeData) {
		for p := range ch {
			marshal, err := json.Marshal(p)
			if err != nil {
				return
			}
			conn.Emit("probe_post", marshal)
		}
	}(conn, ch)
}
