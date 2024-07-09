package probes

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"strconv"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

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
	ReceivedData map[int]TrafficSimData
	ExpectedSeq  int
	AgentID      primitive.ObjectID
}

type ClientStats struct {
	SentPackets    int                `json:"sentPackets"`
	ReceivedAcks   int                `json:"receivedAcks"`
	LastReportTime time.Time          `json:"lastReportTime"`
	ReportInterval time.Duration      `json:"reportInterval"`
	PacketTimes    map[int]PacketTime `json:"-"`
	mu             sync.Mutex
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
		time.Sleep(1 * time.Second)
		ts.Sequence++
		if ts.Sequence > ts.MaxSequence {
			ts.Sequence = 1
		}
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
			ts.ClientStats.SentPackets++
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
				ts.ClientStats.ReceivedAcks++
			}
			ts.ClientStats.mu.Unlock()
			ts.LastResponse = time.Now()
		}
	}
}

func (ts *TrafficSim) reportClientStats() {
	ticker := time.NewTicker(ts.ClientStats.ReportInterval)
	defer ticker.Stop()

	for range ticker.C {
		ts.ClientStats.mu.Lock()
		stats := ts.calculateStats()
		ts.ClientStats.SentPackets = 0
		ts.ClientStats.ReceivedAcks = 0
		ts.ClientStats.PacketTimes = make(map[int]PacketTime)
		ts.ClientStats.LastReportTime = time.Now()
		ts.ClientStats.mu.Unlock()

		ts.DataChan <- ProbeData{
			ProbeID:   ts.Probe,
			Triggered: false,
			Data:      stats,
		}
	}
}

func (ts *TrafficSim) calculateStats() map[string]interface{} {
	var totalRTT, minRTT, maxRTT int64
	lostPackets := 0
	outOfOrder := 0
	lastReceivedTime := int64(0)

	for _, pTime := range ts.ClientStats.PacketTimes {
		if pTime.Received == 0 {
			lostPackets++
		} else {
			rtt := pTime.Received - pTime.Sent
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
			lastReceivedTime = pTime.Received
		}
	}

	avgRTT := int64(0)
	if ts.ClientStats.ReceivedAcks > 0 {
		avgRTT = totalRTT / int64(ts.ClientStats.ReceivedAcks)
	}

	return map[string]interface{}{
		"sentPackets":    ts.ClientStats.SentPackets,
		"receivedAcks":   ts.ClientStats.ReceivedAcks,
		"lostPackets":    lostPackets,
		"outOfOrder":     outOfOrder,
		"averageRTT":     avgRTT,
		"minRTT":         minRTT,
		"maxRTT":         maxRTT,
		"reportInterval": ts.ClientStats.ReportInterval,
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
			ReceivedData: make(map[int]TrafficSimData),
			ExpectedSeq:  1,
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
	connection.ReceivedData[data.Seq] = data

	log.Infof("TrafficSim: Received data from %s: Seq %d", addr.String(), data.Seq)

	ackData := TrafficSimData{
		Sent:     data.Sent,
		Received: time.Now().UnixMilli(),
		Seq:      data.Seq,
	}
	ts.sendACK(conn, addr, ackData)

	if data.Seq == 1 && connection.ExpectedSeq > 1 {
		log.Infof("TrafficSim: Client %s has reset its sequence", addr.String())
		connection.ExpectedSeq = 1
		connection.ReceivedData = make(map[int]TrafficSimData)
	}

	if data.Seq > connection.ExpectedSeq {
		connection.ExpectedSeq = data.Seq + 1
	} else if data.Seq < connection.ExpectedSeq {
		log.Warnf("TrafficSim: Out of sequence packet received. Expected: %d, Got: %d", connection.ExpectedSeq, data.Seq)
	} else {
		connection.ExpectedSeq++
	}

	if len(connection.ReceivedData) >= 10 {
		ts.reportToController(connection)
		connection.ReceivedData = make(map[int]TrafficSimData)
		connection.ExpectedSeq = 1
	}
}

func (ts *TrafficSim) reportToController(connection *Connection) {
	// Implement the actual reporting logic here
	log.Infof("TrafficSim: Reporting stats for client %s", connection.AgentID.Hex())
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
