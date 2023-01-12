package workers

import (
	"encoding/json"
	"github.com/netwatcherio/netwatcher-agent/api"
	"log"
)

func InitQueueWorker(dataChan chan api.CheckData, req api.ApiRequest, client api.Data) {

	go func(ch chan api.CheckData, q api.ApiRequest, c api.Data) {
		var qD []api.CheckData

		for {
			cD := <-ch
			qD = append(qD, cD)
			// make new object??

			m, err := json.Marshal(qD)
			q.Data = string(m)

			print("\n\n\n--------------------------\n" + string(m) + "\n--------------------------\n\n\n")

			err = c.Push(&q)
			if err != nil {
				// handle error on push and save queue for next time??
				log.Println("unable to push apiClient, keeping queue and waiting...")
				continue
			}
			qD = nil
			q.Data = nil
		}
	}(dataChan, req, client)
}
