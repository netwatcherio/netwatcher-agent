package workers

import (
	_ "encoding/json"
	"errors"
	"fmt"
	"github.com/netwatcherio/netwatcher-agent/probes"
	"github.com/showwin/speedtest-go/speedtest"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/sync/syncmap"
	"strconv"
	_ "strconv"
	"strings"
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
		return probes.Probe{}, errors.New("no matching MTR probe found")
	}
	return foundProbe, nil
}

func InitProbeWorker(checkChan chan []probes.Probe, dataChan chan probes.ProbeData, thisAgent primitive.ObjectID) {
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
					startCheckWorker(ad.ID, dataChan, thisAgent)
				} else {
					//checkWorkers.Swap(ad.ID, ad)
					log.Infof("NOT Swapping probe with existing %s", ad.ID.Hex())

					if ad.Type == probes.ProbeType_TRAFFICSIM && ad.Config.Server {
						var allowedAgentsList []primitive.ObjectID

						for _, agent := range ad.Config.Target[1:] {
							allowedAgentsList = append(allowedAgentsList, agent.Agent)
						}

						updateAllowedAgents(trafficSimServer, allowedAgentsList)
					}
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
					// if a check if running, we need to figure out a way to kill the process....? or just let it run?
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

var speedTestRunning = false
var speedTestRetryCount = 0

const speedTestRetryMax = 3

var trafficSimServer *probes.TrafficSim
var trafficSimClients []*probes.TrafficSim

func startCheckWorker(id primitive.ObjectID, dataChan chan probes.ProbeData, thisAgent primitive.ObjectID) {
	go func(i primitive.ObjectID, dC chan probes.ProbeData) {
		for {
			agentCheckW, _ := checkWorkers.Load(i)
			if agentCheckW == nil {
				time.Sleep(5 * time.Second)
			}

			if agentCheckW.(ProbeWorkerS).ToRemove {
				checkWorkers.Delete(i)
				log.Warn("Check with ID " + i.Hex() + " was marked for removal.")
				break
			}

			agentCheck := agentCheckW.(ProbeWorkerS).Probe

			switch agentCheck.Type {
			case probes.ProbeType_TRAFFICSIM:
				checkCfg := agentCheck.Config
				checkAddress := strings.Split(checkCfg.Target[0].Target, ":")

				portNum, err := strconv.Atoi(checkAddress[1])
				if err != nil {
					log.Error(err)
					break
				}

				if agentCheck.Config.Server {
					var allowedAgentsList []primitive.ObjectID

					for _, agent := range agentCheck.Config.Target[1:] {
						allowedAgentsList = append(allowedAgentsList, agent.Agent)
					}

					if trafficSimServer == nil || !trafficSimServer.Running || trafficSimServer.Errored {
						trafficSimServer = &probes.TrafficSim{
							Running:       false,
							Errored:       false,
							IsServer:      true,
							ThisAgent:     thisAgent,
							OtherAgent:    primitive.ObjectID{},
							IPAddress:     checkAddress[0],
							Port:          int64(portNum),
							AllowedAgents: allowedAgentsList,
							Probe:         agentCheck.ID,
						}

						log.Info("Running & starting traffic sim server...")
						trafficSimServer.Running = true
						trafficSimServer.Start()
					} else {
						// Update the allowed agents list dynamically
						updateAllowedAgents(trafficSimServer, allowedAgentsList)
						break
					}
					continue
				} else {
					// Client logic remains the same
					simClient := &probes.TrafficSim{
						Running:    false,
						Errored:    false,
						Conn:       nil,
						ThisAgent:  thisAgent,
						OtherAgent: agentCheck.Config.Target[0].Agent,
						IPAddress:  checkAddress[0],
						Port:       int64(portNum),
						Probe:      agentCheck.ID,
						DataChan:   dC,
					}

					trafficSimClients = append(trafficSimClients, simClient)
					simClient.Running = true
					simClient.Start()
					continue
				}

			case probes.ProbeType_SYSTEMINFO:
				log.Info("SystemInfo: Running system hardware usage test")
				if agentCheck.Config.Interval <= 0 {
					agentCheck.Config.Interval = 1
				}

				mtr, err := probes.SystemInfo()
				if err != nil {
					log.Error(err)
				}

				cD := probes.ProbeData{
					ProbeID: agentCheck.ID,
					Data:    mtr,
				}

				dC <- cD
				time.Sleep(time.Duration(agentCheck.Config.Interval) * time.Minute)
				continue
			case probes.ProbeType_MTR:
				log.Info("MTR: Running test for ", agentCheck.Config.Target[0].Target, "...")
				mtr, err := probes.Mtr(&agentCheck, false)
				if err != nil {
					fmt.Println(err)
				}

				cD := probes.ProbeData{
					ProbeID:   agentCheck.ID,
					Triggered: false,
					Data:      mtr,
				}
				dC <- cD
				time.Sleep(time.Duration(agentCheck.Config.Interval) * time.Minute)
				continue
			/*case probes.ProbeType_RPERF:
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

				//fmt.Println("Sending apiClient to the channel (RPERF) for ", agentCheck.Config.Target, "...")
				dC <- cD
			}
			continue*/
			case probes.ProbeType_SPEEDTEST:
				// todo make this dynamic and on demand
				if !speedTestRunning {
					log.Info("Running speed test for ... ", agentCheck.Config.Target[0].Target)
					if agentCheck.Config.Target[0].Target == "ok" {
						log.Info("SpeedTest: Target is ok, skipping...")
						time.Sleep(10 * time.Second)
						continue
					}
					speedTestResult, err := probes.SpeedTest(&agentCheck)
					if err != nil {
						log.Error(err)
						time.Sleep(30 * time.Second)
						if speedTestRetryCount >= 3 {
							agentCheck.Config.Target[0].Target = "ok"
							speedTestRunning = false
							log.Warn("SpeedTest: Failed to run test after 3 retries, setting target to 'ok'...")
						}
						speedTestRetryCount++
						continue
					}

					speedTestRetryCount = 0

					agentCheck.Config.Target[0].Target = "ok"

					cD := probes.ProbeData{
						ProbeID: agentCheck.ID,
						Data:    speedTestResult,
					}
					speedTestRunning = false

					dC <- cD
				}
				continue
			case probes.ProbeType_SPEEDTEST_SERVERS:
				// todo make this dynamic and on demand
				var speedtestClient = speedtest.New()
				serverList, _ := speedtestClient.FetchServers()
				//targets, _ := serverList.FindServer([]int{})
				// todo ship this off to the backend so we can display "speedtest" servers near the agent, and periodically refresh the options

				cD := probes.ProbeData{
					ProbeID: agentCheck.ID,
					Data:    serverList,
				}

				dC <- cD
				time.Sleep(time.Hour * 12)
				break
			case probes.ProbeType_PING:
				log.Infof("Ping: Running test for %v...", agentCheck.Config.Target[0].Target)

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
				log.Info("NetInfo: Checking networking information...")
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

			// todo other checks like port scans etc.

			default:
				fmt.Println("Unknown type of check...")
				break
			}
			break
		}
	}(id, dataChan)
}

func updateAllowedAgents(server *probes.TrafficSim, newAllowedAgents []primitive.ObjectID) {
	// Use a mutex to ensure thread-safe updates
	server.Mutex.Lock()
	defer server.Mutex.Unlock()

	// Update the allowed agents list
	server.AllowedAgents = newAllowedAgents
	log.Infof("Updated allowed agents for TrafficSim server: %v", newAllowedAgents)
}
