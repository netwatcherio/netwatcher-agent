package workers

import (
	"encoding/json"
	"github.com/netwatcherio/netwatcher-agent/api"
	"log"
)

func InitQueueWorker(dataChan chan api.CheckData, req api.ApiRequest, clientCfg api.ClientConfig) {

	go func(ch chan api.CheckData, q api.ApiRequest, c api.ClientConfig) {
		var qD []api.CheckData

		for {
			cD := <-ch
			qD = append(qD, cD)
			// make new object??

			r := api.ApiRequest{ID: q.ID, PIN: q.PIN}
			client := api.NewClient(c)
			apiClient := api.Data{
				Client: client,
			}

			m, err := json.Marshal(qD)
			r.Data = string(m)

			print("\n\n\n--------------------------\n" + string(m) + "\n--------------------------\n\n\n")

			err = apiClient.Push(&q)
			if err != nil {
				// handle error on push and save queue for next time??
				log.Println("unable to push apiClient, keeping queue and waiting...")
				continue
			}
			//reset the array if the push was successful
			qD = []api.CheckData{}
		}
	}(dataChan, req, clientCfg)
}
