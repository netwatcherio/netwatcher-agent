package probes

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const TrafficSim_ReportSeq = 60
const TrafficSim_DataInterval = 1
const RetryInterval = 5 * time.Second
const PacketTimeout = 2 * time.Second

type TrafficSim struct {
	Running       bool
	Errored       bool
	ThisAgent     primitive.ObjectID
	OtherAgent    primitive.ObjectID
	Conn          *net.UDPConn
	IPAddress     string
	Port          int64
	IsServer      bool
	LastResponse  time.Time
	AllowedAgents []primitive.ObjectID
	Connections   map[primitive.ObjectID]*Connection
	ConnectionsMu sync.RWMutex
	ClientStats   *ClientStats
	Sequence      int
	DataChan      chan ProbeData
	Probe         primitive.ObjectID
	localIP       string
	testComplete  chan bool
	sync.Mutex
}

type Connection struct {
	Addr         *net.UDPAddr
	LastResponse time.Time
	ExpectedSeq  int
	AgentID      primitive.ObjectID
	ClientStats  *ClientStats
}

type ClientStats struct {
	DuplicatePackets int                `json:"duplicatePackets"`
	OutOfSequence    int                `json:"outOfSequence"`
	PacketTimes      map[int]PacketTime `json:"-"`
	LastReportTime   time.Time          `json:"lastReportTime"`
	ReportInterval   time.Duration      `json:"reportInterval"`
	mu               sync.Mutex
}

type PacketTime struct {
	Sent     int64
	Received int64
	TimedOut bool
}

const (
	TrafficSim_HELLO TrafficSimMsgType = "HELLO"
	TrafficSim_ACK   TrafficSimMsgType = "ACK"
	TrafficSim_DATA  TrafficSimMsgType = "DATA"
)

type TrafficSimMsgType string

type TrafficSimMsg struct {
	Type TrafficSimMsgType  `json:"type"`
	Data TrafficSimData     `json:"data"`
	Src  primitive.ObjectID `json:"src"`
	Dst  primitive.ObjectID `json:"dst"`
}

type TrafficSimData struct {
	Sent     int64 `json:"sent"`
	Received int64 `json:"received"`
	Seq      int   `json:"seq"`
}

func (ts *TrafficSim) buildMessage(msgType TrafficSimMsgType, data TrafficSimData) (string, error) {
	msg := TrafficSimMsg{
		Type: msgType,
		Data: data,
		Src:  ts.ThisAgent,
		Dst:  ts.OtherAgent,
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(msgBytes), nil
}

func (ts *TrafficSim) runClient(mtrProbe *Probe) error {
	for ts.Running { // Check Running flag
		currentIP, err := getLocalIP()
		if err != nil {
			log.Errorf("TrafficSim: Failed to get local IP: %v", err)
			if !ts.Running {
				return nil
			}
			time.Sleep(RetryInterval)
			continue
		}

		if ts.localIP != currentIP {
			ts.localIP = currentIP
			log.Infof("TrafficSim: Local IP updated to %s", ts.localIP)
		}

		toAddr, err := net.ResolveUDPAddr("udp4", ts.IPAddress+":"+strconv.Itoa(int(ts.Port)))
		if err != nil {
			log.Errorf("TrafficSim: Could not resolve %v:%d: %v", ts.IPAddress, ts.Port, err)
			if !ts.Running {
				return nil
			}
			time.Sleep(RetryInterval)
			continue
		}

		localAddr, err := net.ResolveUDPAddr("udp4", ts.localIP+":0")
		if err != nil {
			log.Errorf("TrafficSim: Could not resolve local address: %v", err)
			if !ts.Running {
				return nil
			}
			time.Sleep(RetryInterval)
			continue
		}

		conn, err := net.DialUDP("udp4", localAddr, toAddr)
		if err != nil {
			log.Errorf("TrafficSim: Unable to connect to %v:%d: %v", ts.IPAddress, ts.Port, err)
			if !ts.Running {
				return nil
			}
			time.Sleep(RetryInterval)
			continue
		}

		ts.Conn = conn
		ts.ClientStats = &ClientStats{
			LastReportTime: time.Now(),
			ReportInterval: 15 * time.Second,
			PacketTimes:    make(map[int]PacketTime),
		}
		ts.testComplete = make(chan bool, 1)

		if err := ts.sendHello(); err != nil {
			log.Errorf("TrafficSim: Failed to establish connection: %v", err)
			ts.Conn.Close()
			if !ts.Running {
				return nil
			}
			time.Sleep(RetryInterval)
			continue
		}

		log.Infof("TrafficSim: Connection established successfully to %v", ts.OtherAgent.Hex())

		errChan := make(chan error, 3)
		stopChan := make(chan struct{})
		stopOnce := &sync.Once{}

		// Helper function to safely close stopChan
		safeCloseStop := func() {
			stopOnce.Do(func() {
				close(stopChan)
			})
		}

		// Monitor Running flag
		go func() {
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()

			for range ticker.C {
				if !ts.Running {
					safeCloseStop()
					return
				}
			}
		}()

		go ts.runTestCycles(errChan, stopChan, mtrProbe)
		go ts.receiveDataLoop(errChan, stopChan)

		select {
		case err := <-errChan:
			log.Errorf("TrafficSim: Error in client loop: %v", err)
			safeCloseStop()
			ts.Conn.Close()
			if !ts.Running {
				return nil
			}
			time.Sleep(RetryInterval)
		case <-stopChan:
			log.Info("TrafficSim: Client stopped by stopChan")
			ts.Conn.Close()
			return nil
		}
	}

	log.Info("TrafficSim: Client stopped - Running set to false")
	return nil
}

func (ts *TrafficSim) sendHello() error {
	if !ts.Running {
		return fmt.Errorf("trafficSim is not running")
	}

	helloMsg, err := ts.buildMessage(TrafficSim_HELLO, TrafficSimData{Sent: time.Now().UnixMilli()})
	if err != nil {
		return fmt.Errorf("error building hello message: %v", err)
	}

	_, err = ts.Conn.Write([]byte(helloMsg))
	if err != nil {
		return fmt.Errorf("error sending hello message: %v", err)
	}

	msgBuf := make([]byte, 256)
	ts.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, err = ts.Conn.ReadFromUDP(msgBuf)
	if err != nil {
		return fmt.Errorf("error reading hello response: %v", err)
	}

	return nil
}

func (ts *TrafficSim) runTestCycles(errChan chan<- error, stopChan <-chan struct{}, mtrProbe *Probe) {
	for {
		select {
		case <-stopChan:
			return
		default:
			if !ts.Running {
				return
			}

			// Reset for new test cycle
			ts.Mutex.Lock()
			ts.Sequence = 0
			ts.Mutex.Unlock()

			ts.ClientStats.mu.Lock()
			ts.ClientStats.PacketTimes = make(map[int]PacketTime)
			ts.ClientStats.mu.Unlock()

			testStartTime := time.Now()
			packetsInTest := TrafficSim_ReportSeq / TrafficSim_DataInterval

			// Send packets for this test cycle
			for i := 0; i < packetsInTest; i++ {
				select {
				case <-stopChan:
					return
				default:
					if !ts.Running {
						return
					}

					ts.Mutex.Lock()
					ts.Sequence++
					currentSeq := ts.Sequence
					ts.Mutex.Unlock()

					sentTime := time.Now().UnixMilli()
					data := TrafficSimData{Sent: sentTime, Seq: currentSeq}
					dataMsg, err := ts.buildMessage(TrafficSim_DATA, data)
					if err != nil {
						errChan <- fmt.Errorf("error building data message: %v", err)
						return
					}

					_, err = ts.Conn.Write([]byte(dataMsg))
					if err != nil {
						if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
							log.Warn("TrafficSim: Temporary error sending data message:", err)
							continue
						}
						errChan <- fmt.Errorf("error sending data message: %v", err)
						return
					}

					ts.ClientStats.mu.Lock()
					ts.ClientStats.PacketTimes[currentSeq] = PacketTime{Sent: sentTime}
					ts.ClientStats.mu.Unlock()

					// Wait for the interval before sending next packet
					time.Sleep(TrafficSim_DataInterval * time.Second)
				}
			}

			// After sending all packets, wait for responses or timeouts
			log.Debugf("TrafficSim: Finished sending %d packets, waiting for responses...", packetsInTest)

			// Wait for all packets to complete or timeout
			waitStart := time.Now()
			maxWaitTime := PacketTimeout + (500 * time.Millisecond) // Extra buffer

			for time.Since(waitStart) < maxWaitTime {
				if !ts.Running {
					return
				}

				ts.ClientStats.mu.Lock()
				allComplete := true
				now := time.Now().UnixMilli()

				for seq := 1; seq <= packetsInTest; seq++ {
					if pTime, ok := ts.ClientStats.PacketTimes[seq]; ok {
						if pTime.Received == 0 && !pTime.TimedOut {
							// Check if this packet has timed out
							if now-pTime.Sent > int64(PacketTimeout.Milliseconds()) {
								pTime.TimedOut = true
								ts.ClientStats.PacketTimes[seq] = pTime
								log.Debugf("TrafficSim: Packet %d timed out", seq)
							} else {
								allComplete = false
							}
						}
					}
				}
				ts.ClientStats.mu.Unlock()

				if allComplete {
					log.Debugf("TrafficSim: All packets complete or timed out")
					break
				}

				time.Sleep(50 * time.Millisecond)
			}

			// Calculate and report stats
			ts.ClientStats.mu.Lock()
			stats := ts.calculateStats(mtrProbe)
			ts.ClientStats.mu.Unlock()

			if ts.DataChan != nil && ts.Running {
				ts.DataChan <- ProbeData{
					ProbeID:   ts.Probe,
					Triggered: false,
					CreatedAt: time.Now(),
					Data:      stats,
				}
			}

			// Add a small delay before starting the next test cycle
			// This ensures clean separation between tests
			time.Sleep(1 * time.Second)

			log.Debugf("TrafficSim: Test cycle completed in %v", time.Since(testStartTime))
		}
	}
}

func (ts *TrafficSim) receiveDataLoop(errChan chan<- error, stopChan <-chan struct{}) {
	for {
		select {
		case <-stopChan:
			return
		default:
			if !ts.Running {
				return
			}

			msgBuf := make([]byte, 256)
			ts.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			msgLen, _, err := ts.Conn.ReadFromUDP(msgBuf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
					log.Warn("TrafficSim: Temporary error reading from UDP:", err)
					continue
				}
				errChan <- fmt.Errorf("error reading from UDP: %v", err)
				return
			}

			tsMsg := TrafficSimMsg{}
			err = json.Unmarshal(msgBuf[:msgLen], &tsMsg)
			if err != nil {
				log.Error("TrafficSim: Error unmarshalling message:", err)
				continue
			}

			if tsMsg.Type == TrafficSim_ACK {
				data := tsMsg.Data
				seq := data.Seq
				receivedTime := time.Now().UnixMilli()

				ts.ClientStats.mu.Lock()
				if pTime, ok := ts.ClientStats.PacketTimes[seq]; ok && pTime.Received == 0 && !pTime.TimedOut {
					pTime.Received = receivedTime
					ts.ClientStats.PacketTimes[seq] = pTime
					log.Debugf("TrafficSim: Received ACK for packet %d, RTT: %dms", seq, receivedTime-pTime.Sent)
				} else if pTime.TimedOut {
					log.Debugf("TrafficSim: Received late ACK for packet %d (already marked as timed out)", seq)
				}
				ts.ClientStats.mu.Unlock()

				ts.LastResponse = time.Now()
			}
		}
	}
}

func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("no suitable local IP address found")
}

func (ts *TrafficSim) calculateStats(mtrProbe *Probe) map[string]interface{} {
	var totalRTT, minRTT, maxRTT int64
	var rtts []float64
	lostPackets := 0
	outOfOrder := 0
	duplicatePackets := 0
	receivedSequences := []int{}

	// Sort keys to process packets in sequence order
	var keys []int
	for k := range ts.ClientStats.PacketTimes {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, seq := range keys {
		pTime := ts.ClientStats.PacketTimes[seq]
		if pTime.Received == 0 || pTime.TimedOut {
			lostPackets++
			log.Debugf("TrafficSim: Packet %d lost", seq)
			continue
		}

		receivedSequences = append(receivedSequences, seq)
		rtt := pTime.Received - pTime.Sent
		rtts = append(rtts, float64(rtt))
		totalRTT += rtt
		if minRTT == 0 || rtt < minRTT {
			minRTT = rtt
		}
		if rtt > maxRTT {
			maxRTT = rtt
		}
	}

	// Check for out of order packets based on receive time
	for i := 1; i < len(receivedSequences); i++ {
		prevSeq := receivedSequences[i-1]
		currSeq := receivedSequences[i]

		// Packets should be received in order of their sequence numbers
		if currSeq < prevSeq {
			outOfOrder++
			log.Debugf("TrafficSim: Out of order: seq %d received after seq %d", currSeq, prevSeq)
		}
	}

	avgRTT := float64(0)
	stdDevRTT := float64(0)
	if len(rtts) > 0 {
		avgRTT = float64(totalRTT) / float64(len(rtts))

		// Calculate standard deviation
		for _, rtt := range rtts {
			stdDevRTT += math.Pow(rtt-avgRTT, 2)
		}
		stdDevRTT = math.Sqrt(stdDevRTT / float64(len(rtts)))
	}

	totalPackets := len(ts.ClientStats.PacketTimes)
	lossPercentage := float64(0)
	if totalPackets > 0 {
		lossPercentage = (float64(lostPackets) / float64(totalPackets)) * 100
	}

	log.Infof("TrafficSim: Stats - Total: %d, Lost: %d (%.2f%%), Out of Order: %d, Avg RTT: %.2fms",
		totalPackets, lostPackets, lossPercentage, outOfOrder, avgRTT)

	// Trigger MTR if packet loss exceeds threshold percentage
	if totalPackets > 0 && lossPercentage > 5.0 && ts.Running {
		if mtrProbe != nil && len(mtrProbe.Config.Target) > 0 {
			mtr, err := Mtr(mtrProbe, true)
			if err != nil {
				log.Errorf("TrafficSim: MTR error: %v", err)
			}

			if ts.DataChan != nil && ts.Running {
				dC := ProbeData{
					ProbeID:   mtrProbe.ID,
					Triggered: true,
					Data:      mtr,
				}
				log.Infof("TrafficSim: Triggered MTR for %s due to %.2f%% packet loss",
					mtrProbe.Config.Target[0].Target, lossPercentage)
				ts.DataChan <- dC
			}
		}
	}

	return map[string]interface{}{
		"lostPackets":      lostPackets,
		"lossPercentage":   lossPercentage,
		"outOfSequence":    outOfOrder,
		"duplicatePackets": duplicatePackets,
		"averageRTT":       avgRTT,
		"minRTT":           minRTT,
		"maxRTT":           maxRTT,
		"stdDevRTT":        stdDevRTT,
		"totalPackets":     totalPackets,
		"reportTime":       time.Now(),
	}
}

func (ts *TrafficSim) runServer() error {
	addr, err := net.ResolveUDPAddr("udp4", ts.localIP+":"+strconv.Itoa(int(ts.Port)))
	if err != nil {
		return fmt.Errorf("unable to resolve address: %v", err)
	}

	ln, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return fmt.Errorf("unable to listen on %s:%d: %v", ts.localIP, ts.Port, err)
	}
	defer ln.Close()

	log.Infof("TrafficSim: Server listening on %s:%d", ts.localIP, ts.Port)

	ts.Connections = make(map[primitive.ObjectID]*Connection)

	// Set read timeout to check Running flag periodically
	for ts.Running {
		msgBuf := make([]byte, 256)
		ln.SetReadDeadline(time.Now().Add(1 * time.Second))
		msgLen, remoteAddr, err := ln.ReadFromUDP(msgBuf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Check Running flag again
			}
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				log.Warn("TrafficSim: Temporary error reading from UDP:", err)
				continue
			}
			return fmt.Errorf("error reading from UDP: %v", err)
		}

		go ts.handleConnection(ln, remoteAddr, msgBuf[:msgLen])
	}

	log.Info("TrafficSim: Server stopped - Running set to false")
	return nil
}

func (ts *TrafficSim) handleConnection(conn *net.UDPConn, addr *net.UDPAddr, msg []byte) {
	if !ts.Running {
		return
	}

	tsMsg := TrafficSimMsg{}
	err := json.Unmarshal(msg, &tsMsg)
	if err != nil {
		log.Error("TrafficSim: Error unmarshalling message:", err)
		return
	}

	if !ts.isAgentAllowed(tsMsg.Src) {
		log.Error("TrafficSim: Ignoring message from unknown agent:", tsMsg.Src)
		return
	}

	ts.ConnectionsMu.Lock()
	connection, exists := ts.Connections[tsMsg.Src]
	if !exists {
		connection = &Connection{
			Addr:         addr,
			LastResponse: time.Now(),
			AgentID:      tsMsg.Src,
		}
		ts.Connections[tsMsg.Src] = connection
	}
	ts.ConnectionsMu.Unlock()

	switch tsMsg.Type {
	case TrafficSim_HELLO:
		ts.sendACK(conn, addr, TrafficSimData{Sent: time.Now().UnixMilli()})
	case TrafficSim_DATA:
		ts.handleData(conn, addr, tsMsg.Data, connection)
	}
}

func (ts *TrafficSim) sendACK(conn *net.UDPConn, addr *net.UDPAddr, data TrafficSimData) {
	if !ts.Running {
		return
	}

	replyMsg, err := ts.buildMessage(TrafficSim_ACK, data)
	if err != nil {
		log.Error("TrafficSim: Error building reply message:", err)
		return
	}

	_, err = conn.WriteToUDP([]byte(replyMsg), addr)
	if err != nil {
		log.Error("TrafficSim: Error sending ACK:", err)
	}
}

func (ts *TrafficSim) handleData(conn *net.UDPConn, addr *net.UDPAddr, data TrafficSimData, connection *Connection) {
	connection.LastResponse = time.Now()

	log.Debugf("TrafficSim: Received data from %s: Seq %d", addr.String(), data.Seq)

	ackData := TrafficSimData{
		Sent:     data.Sent,
		Received: time.Now().UnixMilli(),
		Seq:      data.Seq,
	}
	ts.sendACK(conn, addr, ackData)
}

func (ts *TrafficSim) reportToController(connection *Connection) {
	// Implement the actual reporting logic here
	// log.Infof("TrafficSim: Reporting stats for client %s", connection.AgentID.Hex())
}

func (ts *TrafficSim) isAgentAllowed(agentID primitive.ObjectID) bool {
	ts.Mutex.Lock()
	defer ts.Mutex.Unlock()

	for _, allowedAgent := range ts.AllowedAgents {
		if allowedAgent == agentID {
			return true
		}
	}
	return false
}

func (ts *TrafficSim) Start(mtrProbe *Probe) {
	defer func() {
		log.Infof("TrafficSim: Start() exiting for probe %s", ts.Probe.Hex())
		ts.Running = false
		if ts.Conn != nil {
			ts.Conn.Close()
		}
	}()

	for ts.Running {
		var err error
		ts.localIP, err = getLocalIP()
		if err != nil {
			log.Errorf("TrafficSim: Failed to get local IP: %v", err)
			if !ts.Running {
				return
			}
			time.Sleep(RetryInterval)
			continue
		}

		if ts.IsServer {
			err = ts.runServer()
		} else {
			err = ts.runClient(mtrProbe)
		}
		if err != nil {
			log.Errorf("TrafficSim: Error occurred: %v. Retrying in %v...", err, RetryInterval)
			if !ts.Running {
				return
			}
			time.Sleep(RetryInterval)
		}
	}
}

// Stop gracefully stops the TrafficSim instance
func (ts *TrafficSim) Stop() {
	log.Infof("TrafficSim: Stopping probe %s", ts.Probe.Hex())
	ts.Mutex.Lock()
	ts.Running = false
	ts.Mutex.Unlock()

	if ts.Conn != nil {
		ts.Conn.Close()
	}
}
