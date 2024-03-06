package workers

import (
	_ "encoding/json"
	"errors"
	"fmt"
	"github.com/netwatcherio/netwatcher-agent/probes"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/sync/syncmap"
	"strconv"
	_ "strconv"
	"time"
	_ "time"
)

type ProbeWorkerS struct {
	Probe    probes.Probe
	ToRemove bool
}

var checkWorkers syncmap.Map

func findMatchingMTRProbe(probe probes.Probe) (probes.Probe, error) {
	var foundProbe probes.Probe
	found := false

	checkWorkers.Range(func(key, value interface{}) bool {
		probeWorker, ok := value.(ProbeWorkerS)
		if !ok {
			// Handle the case where the type assertion fails
			return true // continue iterating
		}

		if probeWorker.Probe.Type == probes.ProbeType_MTR {
			for _, target := range probeWorker.Probe.Config.Target {
				for _, givenTarget := range probe.Config.Target {
					if target.Target == givenTarget.Target {
						foundProbe = probeWorker.Probe
						found = true
						return false // stop iterating
					}
				}
			}
		}
		return true // continue iterating
	})

	if !found {
		return probes.Probe{}, errors.New("no matching probe found")
	}
	return foundProbe, nil
}

func InitProbeWorker(checkChan chan []probes.Probe, dataChan chan probes.ProbeData) {
	go func(aC chan []probes.Probe, dC chan probes.ProbeData) {
		for {
			a := <-aC
			// add to map to continue to update it eventually

			// todo we fetch an array of probes
			// we then need to process the probes to see if they have been changes (eg. targets, etc.)
			// if there haven't been any changes, then we can continue to start/load them
			// if they have changed, then we need to update that pointer to the probe
			// we also need to confirm that if the session with the websocket is still connected

			// currently: checks are ran assuming they will continue to be ran, and only are removed/updated
			// if the actual check is removed. do we keep it like this or allow check modification?

			var newIds []primitive.ObjectID

			for _, ad := range a {

				_, ok := checkWorkers.Load(ad.ID)

				if !ok {
					log.Infof("Starting worker for probe %s", ad.ID.Hex())
					startCheckWorker(ad.ID, dataChan)
				} else {
					//checkWorkers.Swap(ad.ID, ad)
					log.Infof("NOT Swapping probe with existing %s", ad.ID.Hex())
				}

				newIds = append(newIds, ad.ID)

				checkWorkers.Store(ad.ID, ProbeWorkerS{
					Probe:    ad,
					ToRemove: false,
				})
			}

			checkWorkers.Range(func(key any, value any) bool {
				if !contains(newIds, value.(ProbeWorkerS).Probe.ID) {
					v := value.(ProbeWorkerS)
					v.ToRemove = true
					log.Warnf("Probe marked as to be removed %s", v.Probe.ID.Hex())
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

var alreadyRunningTrafficSim = false

func startCheckWorker(id primitive.ObjectID, dataChan chan probes.ProbeData) {
	go func(i primitive.ObjectID, dC chan probes.ProbeData) {
		for {
			agentCheckW, _ := checkWorkers.Load(i)
			if agentCheckW == nil {
				time.Sleep(5 * time.Second)
			}

			if agentCheckW.(ProbeWorkerS).ToRemove {
				checkWorkers.Delete(i)
				fmt.Println("Check with ID " + i.Hex() + " was marked for removal.")
				break
			}

			agentCheck := agentCheckW.(ProbeWorkerS).Probe

			switch agentCheck.Type {
			case probes.ProbeType_SYSTEMINFO:
				log.Info("Running system test")
				if agentCheck.Config.Interval <= 0 {
					agentCheck.Config.Interval = 1
				}

				mtr, err := probes.SystemInfo()
				if err != nil {
					fmt.Println(err)
				}

				cD := probes.ProbeData{
					ProbeID: agentCheck.ID,
					Data:    mtr,
				}

				fmt.Println("Sending apiClient to the channel (Sysinfo) for ", agentCheck.Config.Interval, "...")
				dC <- cD
				fmt.Println("sleeping for " + strconv.Itoa(agentCheck.Config.Interval) + " minutes")
				time.Sleep(time.Duration(agentCheck.Config.Interval) * time.Minute)
				// todo push
				continue
			case probes.ProbeType_MTR:
				log.Info("Running mtr test for ", agentCheck.Config.Target, "...")
				mtr, err := probes.Mtr(&agentCheck, false)
				if err != nil {
					fmt.Println(err)
				}

				/*m, err := json.Marshal(mtr)
				if err != nil {
					fmt.Print(err)
				}*/

				cD := probes.ProbeData{
					ProbeID:   agentCheck.ID,
					Triggered: false,
					Data:      mtr,
				}

				fmt.Println("Sending apiClient to the channel (MTR) for ", agentCheck.Config.Interval, "...")
				dC <- cD
				fmt.Println("sleeping for " + strconv.Itoa(agentCheck.Config.Interval) + " minutes")
				time.Sleep(time.Duration(agentCheck.Config.Interval) * time.Minute)
				// todo push
				continue
			case probes.ProbeType_RPERF:
				// if check says its a server, start a iperf server based on the bind and port provided in target
				//todo
				//make this continue to run, however, make it check if the latest version of the check
				//apiClient contains it, if not, then break out of this thread

				fmt.Println("Running rperf test for ", agentCheck.Config.Target, "...")
				rperf := probes.RPerfResults{}

				if agentCheck.Config.Server {
					err := rperf.Run(&agentCheck)
					if err != nil {
						fmt.Println(err)
						fmt.Println("exiting loop, please check firewall, and recreate check, you may need to reboot")
						time.Sleep(time.Second * 30)
					}
				} else {
					err := rperf.Check(&agentCheck)
					if err != nil {
						fmt.Println(err)
						fmt.Println("something went wrong processing rperf... sleeping for 30 seconds")
						time.Sleep(time.Second * 30)
					}

					cD := probes.ProbeData{
						ProbeID: agentCheck.ID,
						Data:    rperf,
					}

					fmt.Println("Sending apiClient to the channel (RPERF) for ", agentCheck.Config.Target, "...")
					dC <- cD
				}
				continue
			case probes.ProbeType_SPEEDTEST:
				// todo make this dynamic and on demand
				/*if agentCheck.Config.Pending {
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
				}*/
				continue
			case probes.ProbeType_PING:
				log.Info("Running ping test for " + agentCheck.Config.Target[0].Target + "...")

				// todo find target that matches ping host for target field, and run mtr against it
				probe, err := findMatchingMTRProbe(agentCheck)
				if err != nil {
					log.Error(err)
				}

				err = probes.Ping(&agentCheck, dC, probe)
				if err != nil {
					log.Error(err)
					//break
				}

				//todo make this onyl run once, because when it uploads to the server, it will disable it,
				//todo preventing it from being in the configuration after
				//time.Sleep(time.Minute * 5)
				//}
				continue
			case probes.ProbeType_NETWORKINFO:
				fmt.Println("Checking networking information...")
				net, err := probes.NetworkInfo()
				if err != nil {
					fmt.Println(err)
				}

				/*m, err := json.Marshal(net)
				if err != nil {
					fmt.Print(err)
				}*/

				cD := probes.ProbeData{
					ProbeID: agentCheck.ID,
					Data:    net,
				}

				dC <- cD

				// todo make configurable??
				time.Sleep(time.Minute * 10)
				continue
			case probes.ProbeType_TRAFFICSIM:
				if agentCheck.Config.Server {
					probes.InitTrafficSimServer()

					if !alreadyRunningTrafficSim {
						// todo implement call back channel for data / statistics
						log.Info("Running traffic sim server...")
						err := probes.TrafficSimServer(&agentCheck)
						if err != nil {
							fmt.Println(err)
							fmt.Println("exiting loop, please check firewall, and recreate check, you may need to reboot")
							time.Sleep(time.Second * 30)
						}
						alreadyRunningTrafficSim = true
					}
				} else {
					// todo implement call back channel for data / statistics
					probes.TrafficSimClient(&agentCheck)
				}
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
