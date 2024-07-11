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

const TrafficSim_ReportSeq = 15
const TrafficSim_DataInterval = 1

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
	MaxSequence   int
	DataChan      chan ProbeData
	Probe         primitive.ObjectID
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

func (ts *TrafficSim) runClient() {
	toAddr, err := net.ResolveUDPAddr("udp4", ts.IPAddress+":"+strconv.Itoa(int(ts.Port)))
	if err != nil {
		log.Errorf("TrafficSim: Could not resolve %v:%d", ts.IPAddress, ts.Port)
		return
	}

	conn, err := net.DialUDP("udp4", nil, toAddr)
	if err != nil {
		log.Errorf("TrafficSim: Unable to connect to %v:%d", ts.IPAddress, ts.Port)
		return
	}
	defer conn.Close()

	ts.Conn = conn
	ts.ClientStats = &ClientStats{
		LastReportTime: time.Now(),
		ReportInterval: 15 * time.Second,
		PacketTimes:    make(map[int]PacketTime),
	}

	if err := ts.sendHello(); err != nil {
		log.Error("TrafficSim: Failed to establish connection:", err)
		return
	}

	log.Infof("TrafficSim: Connection established successfully to %v", ts.OtherAgent.Hex())

	go ts.sendDataLoop()
	go ts.reportClientStats()
	ts.receiveDataLoop()
}

func (ts *TrafficSim) sendHello() error {
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

func (ts *TrafficSim) sendDataLoop() {
	ts.Sequence = 0
	for {
		time.Sleep(TrafficSim_DataInterval * time.Second)

		ts.Mutex.Lock()
		ts.Sequence++
		ts.Mutex.Unlock()
		sentTime := time.Now().UnixMilli()
		data := TrafficSimData{Sent: sentTime, Seq: ts.Sequence}
		dataMsg, err := ts.buildMessage(TrafficSim_DATA, data)
		if err != nil {
			log.Error("TrafficSim: Error building data message:", err)
			continue
		}

		_, err = ts.Conn.Write([]byte(dataMsg))
		if err != nil {
			log.Error("TrafficSim: Error sending data message:", err)
		} else {
			ts.ClientStats.mu.Lock()
			ts.ClientStats.PacketTimes[ts.Sequence] = PacketTime{Sent: sentTime}
			ts.ClientStats.mu.Unlock()
		}
	}
}

func (ts *TrafficSim) receiveDataLoop() {
	for {
		msgBuf := make([]byte, 256)
		ts.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		msgLen, _, err := ts.Conn.ReadFromUDP(msgBuf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Error("TrafficSim: Timeout: No response received.")
				continue
			}
			log.Error("TrafficSim: Error reading from UDP:", err)
			continue
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
			if pTime, ok := ts.ClientStats.PacketTimes[seq]; ok {
				pTime.Received = receivedTime
				ts.ClientStats.PacketTimes[seq] = pTime
			}
			ts.ClientStats.mu.Unlock()
			ts.LastResponse = time.Now()
		}
	}
}

func (ts *TrafficSim) reportClientStats() {
	ticker := time.NewTicker(TrafficSim_ReportSeq * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ts.ClientStats.mu.Lock()
		stats := ts.calculateStats()
		ts.ClientStats.PacketTimes = make(map[int]PacketTime)
		ts.ClientStats.LastReportTime = time.Now()
		ts.ClientStats.mu.Unlock()
		ts.Sequence = 1

		ts.DataChan <- ProbeData{
			ProbeID:   ts.Probe,
			Triggered: false,
			CreatedAt: time.Now(),
			Data:      stats,
		}
	}
}

func (ts *TrafficSim) calculateStats() map[string]interface{} {
	var totalRTT, minRTT, maxRTT int64
	var rtts []float64
	lostPackets := 0
	outOfOrder := 0
	duplicatePackets := 0
	lastReceivedTime := int64(0)
	lastSeq := 0
	seenPackets := make(map[int]bool)

	// Sort keys to process packets in sequence order
	var keys []int
	for k := range ts.ClientStats.PacketTimes {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, seq := range keys {
		pTime := ts.ClientStats.PacketTimes[seq]
		if pTime.Received == 0 {
			lostPackets++
		} else {
			rtt := pTime.Received - pTime.Sent
			rtts = append(rtts, float64(rtt))
			totalRTT += rtt
			if minRTT == 0 || rtt < minRTT {
				minRTT = rtt
			}
			if rtt > maxRTT {
				maxRTT = rtt
			}
			if pTime.Received < lastReceivedTime {
				outOfOrder++
			}
			if seq < lastSeq {
				outOfOrder++
			}
			lastReceivedTime = pTime.Received
			lastSeq = seq

			// Check for duplicate packets
			if seenPackets[seq] {
				duplicatePackets++
			} else {
				seenPackets[seq] = true
			}
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

	return map[string]interface{}{
		"lostPackets":      lostPackets,
		"outOfSequence":    outOfOrder,
		"duplicatePackets": duplicatePackets,
		"averageRTT":       avgRTT,
		"minRTT":           minRTT,
		"maxRTT":           maxRTT,
		"stdDevRTT":        stdDevRTT,
		"totalPackets":     len(ts.ClientStats.PacketTimes),
		"reportTime":       time.Now().Unix(),
	}
}

func (ts *TrafficSim) runServer() {
	ln, err := net.ListenUDP("udp4", &net.UDPAddr{Port: int(ts.Port)})
	if err != nil {
		log.Errorf("Unable to listen on :%d", ts.Port)
		return
	}
	defer ln.Close()

	log.Infof("Listening on %s:%d", ts.IPAddress, ts.Port)

	ts.Connections = make(map[primitive.ObjectID]*Connection)

	for {
		msgBuf := make([]byte, 256)
		msgLen, addr, err := ln.ReadFromUDP(msgBuf)
		if err != nil {
			log.Error("TrafficSim: Error reading from UDP:", err)
			continue
		}

		go ts.handleConnection(ln, addr, msgBuf[:msgLen])
	}
}

func (ts *TrafficSim) handleConnection(conn *net.UDPConn, addr *net.UDPAddr, msg []byte) {
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

	log.Infof("TrafficSim: Received data from %s: Seq %d", addr.String(), data.Seq)

	ackData := TrafficSimData{
		Sent:     data.Sent,
		Received: time.Now().UnixMilli(),
		Seq:      data.Seq,
	}
	ts.sendACK(conn, addr, ackData)

	/*if data.Seq == 1 && connection.ExpectedSeq > 1 {
		log.Infof("TrafficSim: Client %s has reset its sequence", addr.String())
		connection.ExpectedSeq = 1
	}

	if data.Seq > connection.ExpectedSeq {
		connection.ExpectedSeq = data.Seq + 1
	} else if data.Seq < connection.ExpectedSeq {
		log.Warnf("TrafficSim: Out of sequence packet received. Expected: %d, Got: %d", connection.ExpectedSeq, data.Seq)
	} else {
		connection.ExpectedSeq++
	}*/

	/*if len(connection.ReceivedData) >= 10 {
		ts.reportToController(connection)
		connection.ReceivedData = make(map[int]TrafficSimData)
		connection.ExpectedSeq = 1
	}*/
}

func (ts *TrafficSim) reportToController(connection *Connection) {
	// Implement the actual reporting logic here
	// log.Infof("TrafficSim: Reporting stats for client %s", connection.AgentID.Hex())
}

func (ts *TrafficSim) isAgentAllowed(agentID primitive.ObjectID) bool {
	for _, allowedAgent := range ts.AllowedAgents {
		if allowedAgent == agentID {
			return true
		}
	}
	return false
}

func (ts *TrafficSim) Start() {
	if ts.IsServer {
		ts.runServer()
	} else {
		ts.runClient()
	}
}
