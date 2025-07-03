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
	"sync"
	"time"
	_ "time"
)

type ProbeWorkerS struct {
	Probe     probes.Probe
	ToRemove  bool
	StopChan  chan struct{}   // For stopping TrafficSim instances
	WaitGroup *sync.WaitGroup // For waiting for cleanup
}

var checkWorkers syncmap.Map

// Track TrafficSim instances
var trafficSimServer *probes.TrafficSim
var trafficSimServerMutex sync.Mutex
var trafficSimClients map[primitive.ObjectID]*probes.TrafficSim
var trafficSimClientsMutex sync.Mutex

func init() {
	trafficSimClients = make(map[primitive.ObjectID]*probes.TrafficSim)
}

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
					if strings.Contains(givenTarget.Target, ":") {
						tt := strings.Split(givenTarget.Target, ":")
						if tt[0] == target.Target {
							foundProbe = probeWorker.Probe
							found = true
							return false // stop iterating
						}
					}

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

func trafficSimConfigChanged(oldProbe, newProbe probes.Probe) bool {
	// Check if target address/port changed
	if len(oldProbe.Config.Target) > 0 && len(newProbe.Config.Target) > 0 {
		if oldProbe.Config.Target[0].Target != newProbe.Config.Target[0].Target {
			return true
		}
		if oldProbe.Config.Target[0].Agent != newProbe.Config.Target[0].Agent {
			return true
		}
	}

	// Check if server flag changed
	if oldProbe.Config.Server != newProbe.Config.Server {
		return true
	}

	return false
}

func stopTrafficSim(probeID primitive.ObjectID, isServer bool) {
	if isServer {
		trafficSimServerMutex.Lock()
		defer trafficSimServerMutex.Unlock()

		if trafficSimServer != nil {
			log.Infof("Stopping TrafficSim server for probe %s", probeID.Hex())
			// Use Stop() if available, otherwise manually stop
			if trafficSimServer.Running {
				trafficSimServer.Running = false
				if trafficSimServer.Conn != nil {
					trafficSimServer.Conn.Close()
				}
			}
			trafficSimServer = nil
		}
	} else {
		trafficSimClientsMutex.Lock()
		defer trafficSimClientsMutex.Unlock()

		if client, exists := trafficSimClients[probeID]; exists {
			log.Infof("Stopping TrafficSim client for probe %s", probeID.Hex())
			// Use Stop() if available, otherwise manually stop
			if client.Running {
				client.Running = false
				if client.Conn != nil {
					client.Conn.Close()
				}
			}
			delete(trafficSimClients, probeID)
		}
	}
}

func InitProbeWorker(checkChan chan []probes.Probe, dataChan chan probes.ProbeData, thisAgent primitive.ObjectID) {
	go func(aC chan []probes.Probe, dC chan probes.ProbeData) {
		for {
			a := <-aC

			var newIds []primitive.ObjectID

			for _, ad := range a {
				existingWorker, ok := checkWorkers.Load(ad.ID)

				if !ok {
					log.Infof("Starting NEW worker for probe %s (type: %s)", ad.ID.Hex(), ad.Type)
					// Store the probe first with new StopChan
					stopChan := make(chan struct{})
					wg := &sync.WaitGroup{}
					checkWorkers.Store(ad.ID, ProbeWorkerS{
						Probe:     ad,
						ToRemove:  false,
						StopChan:  stopChan,
						WaitGroup: wg,
					})
					startCheckWorker(ad.ID, dataChan, thisAgent)
				} else {
					oldProbeWorker := existingWorker.(ProbeWorkerS)

					// Check if TrafficSim config changed
					if ad.Type == probes.ProbeType_TRAFFICSIM && trafficSimConfigChanged(oldProbeWorker.Probe, ad) {
						log.Infof("TrafficSim probe %s configuration changed, restarting", ad.ID.Hex())

						// Stop the old instance
						if oldProbeWorker.StopChan != nil {
							close(oldProbeWorker.StopChan)
						}

						// Wait for worker to finish
						if oldProbeWorker.WaitGroup != nil {
							log.Debugf("Waiting for old TrafficSim worker %s to stop...", ad.ID.Hex())
							oldProbeWorker.WaitGroup.Wait()
							log.Debugf("Old TrafficSim worker %s stopped", ad.ID.Hex())
						}

						// Force stop TrafficSim instance
						stopTrafficSim(ad.ID, oldProbeWorker.Probe.Config.Server)

						// Wait a bit more for complete cleanup
						time.Sleep(1 * time.Second)

						// Store the updated probe with new StopChan
						newStopChan := make(chan struct{})
						newWg := &sync.WaitGroup{}
						checkWorkers.Store(ad.ID, ProbeWorkerS{
							Probe:     ad,
							ToRemove:  false,
							StopChan:  newStopChan,
							WaitGroup: newWg,
						})

						// Start new worker
						log.Infof("Starting UPDATED worker for probe %s", ad.ID.Hex())
						startCheckWorker(ad.ID, dataChan, thisAgent)
					} else if ad.Type == probes.ProbeType_TRAFFICSIM && ad.Config.Server {
						// Just update allowed agents for server
						var allowedAgentsList []primitive.ObjectID
						for _, agent := range ad.Config.Target[1:] {
							allowedAgentsList = append(allowedAgentsList, agent.Agent)
						}

						trafficSimServerMutex.Lock()
						if trafficSimServer != nil {
							updateAllowedAgents(trafficSimServer, allowedAgentsList)
						}
						trafficSimServerMutex.Unlock()

						// Update the probe data
						oldProbeWorker.Probe = ad
						checkWorkers.Store(ad.ID, oldProbeWorker)
					} else {
						// Update the probe data for non-TrafficSim types
						oldProbeWorker.Probe = ad
						checkWorkers.Store(ad.ID, oldProbeWorker)
					}
				}

				newIds = append(newIds, ad.ID)
			}

			// Mark probes for removal
			checkWorkers.Range(func(key any, value any) bool {
				probeWorker := value.(ProbeWorkerS)
				if !contains(newIds, probeWorker.Probe.ID) {
					probeWorker.ToRemove = true
					checkWorkers.Store(key, probeWorker)
					log.Warnf("Probe marked as to be removed %s", probeWorker.Probe.ID.Hex())

					// Stop TrafficSim if it's running
					if probeWorker.Probe.Type == probes.ProbeType_TRAFFICSIM {
						if probeWorker.StopChan != nil {
							close(probeWorker.StopChan)
						}

						// Wait for cleanup
						if probeWorker.WaitGroup != nil {
							probeWorker.WaitGroup.Wait()
						}

						stopTrafficSim(probeWorker.Probe.ID, probeWorker.Probe.Config.Server)
					}
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

func startCheckWorker(id primitive.ObjectID, dataChan chan probes.ProbeData, thisAgent primitive.ObjectID) {
	go func(i primitive.ObjectID, dC chan probes.ProbeData) {
		// Get the worker and increment its WaitGroup
		var wg *sync.WaitGroup
		var stopChan chan struct{}

		if agentCheckW, exists := checkWorkers.Load(i); exists {
			if pw, ok := agentCheckW.(ProbeWorkerS); ok {
				if pw.WaitGroup != nil {
					wg = pw.WaitGroup
					wg.Add(1)
					defer wg.Done()
				}
				stopChan = pw.StopChan
			}
		} else {
			log.Warnf("Probe %s not found when starting worker", i.Hex())
			return
		}

		for {
			agentCheckW, exists := checkWorkers.Load(i)
			if !exists || agentCheckW == nil {
				log.Warnf("Probe %s not found, exiting worker", i.Hex())
				return
			}

			probeWorker := agentCheckW.(ProbeWorkerS)

			if probeWorker.ToRemove {
				checkWorkers.Delete(i)
				log.Warn("Check with ID " + i.Hex() + " was marked for removal.")
				break
			}

			agentCheck := probeWorker.Probe

			switch agentCheck.Type {
			case probes.ProbeType_TRAFFICSIM:
				// Check for stop signal
				if stopChan != nil {
					select {
					case <-stopChan:
						log.Infof("TrafficSim worker %s received stop signal", i.Hex())
						return
					default:
					}
				}

				checkCfg := agentCheck.Config
				checkAddress := strings.Split(checkCfg.Target[0].Target, ":")

				portNum, err := strconv.Atoi(checkAddress[1])
				if err != nil {
					log.Error(err)
					break
				}

				probe, err := findMatchingMTRProbe(agentCheck)
				if err != nil {
					log.Error(err)
				}

				if agentCheck.Config.Server {
					var allowedAgentsList []primitive.ObjectID

					for _, agent := range agentCheck.Config.Target[1:] {
						allowedAgentsList = append(allowedAgentsList, agent.Agent)
					}

					trafficSimServerMutex.Lock()
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
						trafficSimServerMutex.Unlock()

						// Start server in a separate goroutine so we can monitor stop signal
						go trafficSimServer.Start(nil)

						// Monitor for stop signal
						if stopChan != nil {
							<-stopChan
							log.Infof("Stopping TrafficSim server %s", i.Hex())
							stopTrafficSim(agentCheck.ID, true)
						}
						return
					} else {
						// Update the allowed agents list dynamically
						updateAllowedAgents(trafficSimServer, allowedAgentsList)
						trafficSimServerMutex.Unlock()

						// Continue monitoring for changes
						time.Sleep(5 * time.Second)
						continue
					}
				} else {
					// Client logic
					trafficSimClientsMutex.Lock()

					// Check if this client already exists
					if existingClient, exists := trafficSimClients[agentCheck.ID]; exists && existingClient.Running {
						trafficSimClientsMutex.Unlock()
						time.Sleep(5 * time.Second)
						continue
					}

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

					trafficSimClients[agentCheck.ID] = simClient
					simClient.Running = true
					trafficSimClientsMutex.Unlock()

					log.Infof("Starting TrafficSim client for probe %s to %s:%d",
						agentCheck.ID.Hex(), checkAddress[0], portNum)

					// Start client in a separate goroutine so we can monitor stop signal
					go simClient.Start(&probe)

					// Monitor for stop signal
					if stopChan != nil {
						<-stopChan
						log.Infof("Stopping TrafficSim client %s", i.Hex())
						stopTrafficSim(agentCheck.ID, false)
						// Wait a moment to ensure cleanup completes
						time.Sleep(200 * time.Millisecond)
					}
					return
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

			case probes.ProbeType_SPEEDTEST:
				// todo make this dynamic and on demand
				if !speedTestRunning {
					log.Info("Running speed test for ... ", agentCheck.Config.Target[0].Target)
					if agentCheck.Config.Target[0].Target == "ok" {
						log.Info("SpeedTest: Target is ok, skipping...")
						time.Sleep(10 * time.Second)
						continue
					}
					speedTestRunning = true
					speedTestResult, err := probes.SpeedTest(&agentCheck)
					if err != nil {
						log.Error(err)
						speedTestRunning = false
						time.Sleep(30 * time.Second)
						if speedTestRetryCount >= 3 {
							agentCheck.Config.Target[0].Target = "ok"
							log.Warn("SpeedTest: Failed to run test after 3 retries, setting target to 'ok'...")
						}
						speedTestRetryCount++
						continue
					}

					speedTestRetryCount = 0
					speedTestRunning = false

					agentCheck.Config.Target[0].Target = "ok"

					cD := probes.ProbeData{
						ProbeID: agentCheck.ID,
						Data:    speedTestResult,
					}

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
