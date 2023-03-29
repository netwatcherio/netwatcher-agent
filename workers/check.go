package workers

import (
	"encoding/json"
	_ "encoding/json"
	"fmt"
	"github.com/netwatcherio/netwatcher-agent/api"
	"github.com/netwatcherio/netwatcher-agent/checks"
	_ "github.com/netwatcherio/netwatcher-agent/checks"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/sync/syncmap"
	"strconv"
	_ "strconv"
	"time"
	_ "time"
)

type CheckWorkerS struct {
	Check    api.AgentCheck
	ToRemove bool
}

var checkWorkers syncmap.Map

func InitCheckWorker(checkChan chan []api.AgentCheck, dataChan chan api.CheckData) {
	go func(aC chan []api.AgentCheck, dC chan api.CheckData) {
		for {
			a := <-aC
			// add to map to continue to update it eventually

			var newIds []primitive.ObjectID

			// loop through new received config agents and add them to the check workers
			for _, ad := range a {

				_, ok := checkWorkers.Load(ad.ID)

				if !ok {
					startCheckWorker(ad.ID, dataChan)
				}

				newIds = append(newIds, ad.ID)

				checkWorkers.Store(ad.ID, CheckWorkerS{
					Check:    ad,
					ToRemove: false,
				})
			}

			checkWorkers.Range(func(key any, value any) bool {
				if !contains(newIds, value.(CheckWorkerS).Check.ID) {
					v := value.(CheckWorkerS)
					v.ToRemove = true
				}
				return true
			})
		}
	}(checkChan, dataChan)
}

func contains(ids []primitive.ObjectID, id primitive.ObjectID) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func startCheckWorker(id primitive.ObjectID, dataChan chan api.CheckData) {
	go func(i primitive.ObjectID, dC chan api.CheckData) {
		for {
			agentCheckW, _ := checkWorkers.Load(i)
			if agentCheckW == nil {
				time.Sleep(5 * time.Second)
			}

			if agentCheckW.(CheckWorkerS).ToRemove {
				checkWorkers.Delete(i)
				fmt.Println("Check with ID " + i.Hex() + " was marked for removal.")
				break
			}

			agentCheck := agentCheckW.(CheckWorkerS).Check

			switch agentCheck.Type {
			case api.CtMtr:
				fmt.Println("Running mtr test for ", agentCheck.Target, "...")
				mtr, err := checks.Mtr(&agentCheck, false)
				if err != nil {
					fmt.Println(err)
				}

				m, err := json.Marshal(mtr)
				if err != nil {
					fmt.Print(err)
				}

				cD := api.CheckData{
					Target:    agentCheck.Target,
					CheckID:   agentCheck.ID,
					AgentID:   agentCheck.AgentID,
					Triggered: mtr.Triggered,
					Result:    string(m),
					Type:      api.CtMtr,
				}

				fmt.Println("Sending apiClient to the channel (MTR) for ", agentCheck.Interval, "...")
				dC <- cD
				fmt.Println("sleeping for " + strconv.Itoa(agentCheck.Interval) + " minutes")
				time.Sleep(time.Duration(agentCheck.Interval) * time.Minute)
				// todo push
				continue
			case api.CtRperf:
				// if check says its a server, start a iperf server based on the bind and port provided in target
				//todo
				//make this continue to run, however, make it check if the latest version of the check
				//apiClient contains it, if not, then break out of this thread

				fmt.Println("Running rperf test for ", agentCheck.Target, "...")
				rperf := checks.RPerfResults{}

				if agentCheck.Server {
					err := rperf.Run(&agentCheck)
					if err != nil {
						fmt.Println(err)
						fmt.Println("exiting loop, please check firewall, and recreate check, you may need to reboot")
						time.Sleep(time.Second * 30)
						break
					}
				} else {
					err := rperf.Check(&agentCheck)
					if err != nil {
						fmt.Println(err)
						fmt.Println("something went wrong processing rperf... sleeping for 30 seconds")
						time.Sleep(time.Second * 30)
					}

					m, err := json.Marshal(rperf)
					if err != nil {
						fmt.Print(err)
					}

					cD := api.CheckData{
						Target:  agentCheck.Target,
						CheckID: agentCheck.ID,
						AgentID: agentCheck.AgentID,
						Result:  string(m),
						Type:    api.CtRperf,
					}

					fmt.Println("Sending apiClient to the channel (RPERF) for ", agentCheck.Target, "...")
					dC <- cD
				}
				continue
			case api.CtSpeedtest:
				if agentCheck.Pending {
					fmt.Println("Running speed test...")
					speedtest, err := checks.SpeedTest(&agentCheck)
					if err != nil {
						fmt.Println(err)
						return
					}

					m, err := json.Marshal(speedtest)
					if err != nil {
						fmt.Print(err)
					}

					cD := api.CheckData{
						Target:  agentCheck.Target,
						CheckID: agentCheck.ID,
						AgentID: agentCheck.AgentID,
						Result:  string(m),
						Type:    api.CtSpeedtest,
					}

					dC <- cD

					//todo make this onyl run once, because when it uploads to the server, it will disable it,
					//todo preventing it from being in the configuration after
					//time.Sleep(time.Minute * 5)
					//}
				}
				continue
			case api.CtPing:
				fmt.Println("Running ping test for " + agentCheck.Target + "...")
				pingC := make(chan checks.PingResult)
				go func(ac api.AgentCheck, ch chan checks.PingResult) {
					checks.Ping(&ac, ch)
				}(agentCheck, pingC)

				for {
					ping := <-pingC

					m, err := json.Marshal(ping)
					if err != nil {
						fmt.Print(err)
					}

					cD := api.CheckData{
						Target:  agentCheck.Target,
						CheckID: agentCheck.ID,
						AgentID: agentCheck.AgentID,
						Result:  string(m),
						Type:    api.CtPing,
					}

					dC <- cD
					break
				}

				//todo make this onyl run once, because when it uploads to the server, it will disable it,
				//todo preventing it from being in the configuration after
				//time.Sleep(time.Minute * 5)
				//}
				continue
			case api.CtNetinfo:
				fmt.Println("Checking networking information...")
				net, err := checks.NetworkInfo()
				if err != nil {
					fmt.Println(err)
				}

				m, err := json.Marshal(net)
				if err != nil {
					fmt.Print(err)
				}

				cD := api.CheckData{
					Target:  agentCheck.Target,
					CheckID: agentCheck.ID,
					AgentID: agentCheck.AgentID,
					Result:  string(m),
					Type:    api.CtNetinfo,
				}

				dC <- cD

				// todo make configurable??
				time.Sleep(time.Minute * 10)
				continue

			// todo other checks like port scans etc.

			default:
				fmt.Println("Unknown type of check...")
				break
			}
			break
		}
	}(id, dataChan)
}
