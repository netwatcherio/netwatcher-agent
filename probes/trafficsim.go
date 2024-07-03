package probes

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TrafficSim struct {
	Running          bool
	Errored          bool
	DataSend         chan string
	DataReceive      chan string
	ThisAgent        primitive.ObjectID
	OtherAgent       primitive.ObjectID
	Conn             *net.UDPConn
	IPAddress        string
	Port             int64
	IsServer         bool
	LastResponse     time.Time
	Registered       bool
	AllowedAgents    []primitive.ObjectID
	Connections      map[string]*Connection
	ConnectionsMu    sync.RWMutex
	ClientStats      *ClientStats
	Sequence         int
	ExpectedSequence int
	DataChan         *chan ProbeData
	Probe            primitive.ObjectID
}

type Connection struct {
	Addr         *net.UDPAddr
	LastResponse time.Time
	LostPackets  int
	ReceivedData []TrafficSimData
}

type ClientStats struct {
	SentPackets    int           `json:"sentPackets,omitempty"`
	ReceivedAcks   int           `json:"receivedAcks,omitempty"`
	LostPackets    int           `json:"lostPackets,omitempty"`
	OutOfSequence  int           `json:"outOfSequence,omitempty"`
	LastReportTime time.Time     `json:"lastReportTime"`
	AverageRTT     int64         `json:"averageRTT,omitempty"` // in milliseconds
	TotalRTT       int64         `json:"totalRTT,omitempty"`   // in milliseconds
	MinRTT         int64         `json:"minRTT,omitempty"`     // in milliseconds
	MaxRTT         int64         `json:"maxRTT,omitempty"`     // in milliseconds
	ReportInterval time.Duration `json:"reportInterval,omitempty"`
	mu             sync.Mutex
}

const (
	TrafficSim_HELLO TrafficSimMsgType = "HELLO"
	TrafficSim_ACK   TrafficSimMsgType = "ACK"
	TrafficSim_DATA  TrafficSimMsgType = "DATA"
)

type TrafficSimMsgType string

type TrafficSimMsg struct {
	Type TrafficSimMsgType  `json:"type,omitempty"`
	Data TrafficSimData     `json:"data,omitempty"`
	Src  primitive.ObjectID `json:"src,omitempty"`
	Dst  primitive.ObjectID `json:"dst,omitempty"`
}

type TrafficSimData struct {
	Sent     int64 `json:"sent"`     // Unix timestamp in milliseconds
	Received int64 `json:"received"` // Unix timestamp in milliseconds
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

func (ts *TrafficSim) runClient(dC chan ProbeData) {
	toAddr, err := net.ResolveUDPAddr("udp4", ts.IPAddress+":"+strconv.Itoa(int(ts.Port)))
	if err != nil {
		fmt.Printf("Could not resolve %s:%d\n", ts.IPAddress, ts.Port)
		return
	}

	fmt.Printf("Trying to connect to %s:%d\n", ts.IPAddress, ts.Port)

	conn, err := net.DialUDP("udp4", nil, toAddr)
	err = conn.SetWriteBuffer(4096)
	err = conn.SetReadBuffer(4096)
	if err != nil {
		fmt.Printf("Unable to connect to %s:%d\n", ts.IPAddress, ts.Port)
		return
	}
	defer conn.Close()

	// define client stat interval
	ts.Conn = conn
	ts.ClientStats = &ClientStats{
		LastReportTime: time.Now(),
		ReportInterval: 15 * time.Second,
	}

	if err := ts.sendHello(); err != nil {
		fmt.Println("Failed to establish connection:", err)
		return
	}

	fmt.Println("Connection established successfully")

	go ts.sendDataLoop()
	go ts.reportClientStats(dC)
	ts.receiveDataLoop()
}

func (ts *TrafficSim) sendHello() error {
	helloMsg, err := ts.buildMessage(TrafficSim_HELLO, TrafficSimData{Sent: time.Now().UnixMilli()})
	if err != nil {
		return fmt.Errorf("error building hello message: %w", err)
	}

	_, err = ts.Conn.Write([]byte(helloMsg))
	if err != nil {
		return fmt.Errorf("error sending hello message: %w", err)
	}

	msgBuf := make([]byte, 1024)
	ts.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, err = ts.Conn.ReadFromUDP(msgBuf)
	if err != nil {
		return fmt.Errorf("error reading hello response: %w", err)
	}

	return nil
}

func (ts *TrafficSim) sendDataLoop() {
	ts.Sequence = 0
	for {
		time.Sleep(1 * time.Second)
		ts.Sequence++
		sentTime := time.Now().UnixMilli()
		data := TrafficSimData{Sent: sentTime, Seq: ts.Sequence}
		dataMsg, err := ts.buildMessage(TrafficSim_DATA, data)
		if err != nil {
			fmt.Println("Error building data message:", err)
			continue
		}

		_, err = ts.Conn.Write([]byte(dataMsg))
		if err != nil {
			fmt.Println("Error sending data message:", err)
			ts.ClientStats.mu.Lock()
			ts.ClientStats.LostPackets++
			ts.ClientStats.mu.Unlock()
		} else {
			ts.ClientStats.mu.Lock()
			ts.ClientStats.SentPackets++
			ts.ClientStats.mu.Unlock()
		}
	}
}

func (ts *TrafficSim) receiveDataLoop() {
	ts.ExpectedSequence = 1
	for {
		msgBuf := make([]byte, 1024)
		ts.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		msgLen, _, err := ts.Conn.ReadFromUDP(msgBuf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				fmt.Println("Timeout: No response received.")
				ts.ClientStats.mu.Lock()
				ts.ClientStats.LostPackets++
				ts.ClientStats.mu.Unlock()
				continue
			}
			fmt.Println("Error reading from UDP:", err)
			ts.ClientStats.mu.Lock()
			ts.ClientStats.LostPackets++
			ts.ExpectedSequence++
			ts.ClientStats.mu.Unlock()
			continue
		}

		tsMsg := TrafficSimMsg{}
		err = json.Unmarshal(msgBuf[:msgLen], &tsMsg)
		if err != nil {
			fmt.Println("Error unmarshalling message:", err)
			continue
		}

		if tsMsg.Type == TrafficSim_ACK {
			data := tsMsg.Data
			seq := data.Seq
			receivedTime := time.Now().UnixMilli()
			rtt := (receivedTime - data.Sent) + (data.Received - data.Sent)

			// Ensure RTT is non-negative
			if rtt < 0 {
				rtt = 0
			}

			ts.ClientStats.mu.Lock()
			ts.ClientStats.ReceivedAcks++
			ts.ClientStats.TotalRTT += rtt
			ts.ClientStats.AverageRTT = ts.ClientStats.TotalRTT / int64(ts.ClientStats.ReceivedAcks)

			// Update min and max RTT
			if ts.ClientStats.MinRTT == 0 || rtt < ts.ClientStats.MinRTT {
				ts.ClientStats.MinRTT = rtt
			}
			if rtt > ts.ClientStats.MaxRTT {
				ts.ClientStats.MaxRTT = rtt
			}

			if seq != ts.ExpectedSequence {
				fmt.Printf("Out of sequence ACK received. Expected: %d, Got: %d\n", ts.ExpectedSequence, seq)
				ts.ClientStats.OutOfSequence++
			} else {
				fmt.Printf("Received ACK: Seq %d, RTT: %.2f ms\n", seq, float64(rtt))
				ts.ExpectedSequence++
			}
			ts.ClientStats.mu.Unlock()

			ts.LastResponse = time.Now()
		}
	}
}

func (ts *TrafficSim) reportClientStats(dC chan ProbeData) {
	ticker := time.NewTicker(ts.ClientStats.ReportInterval)
	defer ticker.Stop()

	for range ticker.C {
		ts.ClientStats.mu.Lock()

		// Create a new struct with only the data we need, excluding the mutex
		statsCopy := struct {
			SentPackets    int
			ReceivedAcks   int
			LostPackets    int
			OutOfSequence  int
			AverageRTT     int64
			MinRTT         int64
			MaxRTT         int64
			ReportInterval time.Duration
			LastReportTime time.Time
		}{
			SentPackets:    ts.ClientStats.SentPackets,
			ReceivedAcks:   ts.ClientStats.ReceivedAcks,
			LostPackets:    ts.ClientStats.LostPackets,
			OutOfSequence:  ts.ClientStats.OutOfSequence,
			AverageRTT:     ts.ClientStats.AverageRTT,
			MinRTT:         ts.ClientStats.MinRTT,
			MaxRTT:         ts.ClientStats.MaxRTT,
			LastReportTime: time.Now(),
			ReportInterval: ts.ClientStats.ReportInterval,
		}

		// Reset the stats
		ts.ClientStats.SentPackets = 0
		ts.ClientStats.ReceivedAcks = 0
		ts.ClientStats.LostPackets = 0
		ts.ClientStats.OutOfSequence = 0
		ts.ClientStats.TotalRTT = 0
		ts.ClientStats.AverageRTT = 0
		ts.ClientStats.MinRTT = 0
		ts.ClientStats.MaxRTT = 0

		ts.ClientStats.mu.Unlock()

		// Print the stats
		fmt.Printf("\n--- Client Connection Statistics ---\n")
		fmt.Printf("Sent Packets: %d\n", statsCopy.SentPackets)
		fmt.Printf("Received ACKs: %d\n", statsCopy.ReceivedAcks)
		fmt.Printf("Lost Packets: %d\n", statsCopy.LostPackets)
		fmt.Printf("Out of Sequence: %d\n", statsCopy.OutOfSequence)
		fmt.Printf("Average RTT: %d ms\n", statsCopy.AverageRTT)
		fmt.Printf("Min RTT: %d ms\n", statsCopy.MinRTT)
		fmt.Printf("Max RTT: %d ms\n", statsCopy.MaxRTT)
		fmt.Printf("-----------------------------------\n\n")

		// Send the data to the channel
		cD := ProbeData{
			ProbeID:   ts.Probe,
			Triggered: false,
			Data:      statsCopy,
		}
		dC <- cD

		// Reset sequence numbers outside of the lock
		ts.Sequence = 0
		ts.ExpectedSequence = 1
	}
}

func (ts *TrafficSim) runServer() {
	ln, err := net.ListenUDP("udp4", &net.UDPAddr{Port: int(ts.Port)})
	err = ln.SetWriteBuffer(4096)
	err = ln.SetReadBuffer(4096)
	if err != nil {
		fmt.Printf("Unable to listen on :%d\n", ts.Port)
		return
	}
	defer ln.Close()

	fmt.Printf("Listening on :%d\n", ts.Port)

	ts.Connections = make(map[string]*Connection)

	for {
		ts.listenForConnections(ln)
	}
}

func (ts *TrafficSim) listenForConnections(ln *net.UDPConn) {
	msgBuf := make([]byte, 1024)
	ln.SetReadDeadline(time.Now().Add(5 * time.Second))
	rcvLen, addr, err := ln.ReadFromUDP(msgBuf)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			fmt.Println("Read timeout: no data received, continuing to listen...")
			return
		}
		fmt.Println("Error reading from UDP:", err)
		return
	}

	go ts.handleConnection(ln, addr, msgBuf[:rcvLen])
}

func (ts *TrafficSim) handleConnection(conn *net.UDPConn, addr *net.UDPAddr, msg []byte) {
	addrKey := addr.String()

	ts.ConnectionsMu.Lock()
	connection, exists := ts.Connections[addrKey]
	if !exists {
		connection = &Connection{
			Addr:         addr,
			LastResponse: time.Now(),
			LostPackets:  0,
			ReceivedData: []TrafficSimData{},
		}
		ts.Connections[addrKey] = connection
	}
	ts.ConnectionsMu.Unlock()

	tsMsg := TrafficSimMsg{}
	err := json.Unmarshal(msg, &tsMsg)
	if err != nil {
		fmt.Println("Error unmarshalling message:", err)
		return
	}

	if !ts.isAgentAllowed(tsMsg.Src) {
		fmt.Println("Ignoring message from unknown agent:", tsMsg.Src)
		return
	}

	switch tsMsg.Type {
	case TrafficSim_HELLO:
		ts.sendACK(conn, addr, TrafficSimData{Sent: time.Now().UnixMilli()})
	case TrafficSim_DATA:
		ts.handleData(conn, addr, tsMsg.Data)
	}
}

func (ts *TrafficSim) sendACK(conn *net.UDPConn, addr *net.UDPAddr, data TrafficSimData) {
	replyMsg, err := ts.buildMessage(TrafficSim_ACK, data)
	if err != nil {
		fmt.Println("Error building reply message:", err)
		return
	}

	_, err = conn.WriteToUDP([]byte(replyMsg), addr)
	if err != nil {
		fmt.Println("Error sending ACK:", err)
	}
}

func (ts *TrafficSim) handleData(conn *net.UDPConn, addr *net.UDPAddr, data TrafficSimData) {
	addrKey := addr.String()

	ts.ConnectionsMu.Lock()
	defer ts.ConnectionsMu.Unlock()

	connection := ts.Connections[addrKey]
	connection.LastResponse = time.Now()
	connection.ReceivedData = append(connection.ReceivedData, data)

	fmt.Printf("Received data from %s: Seq %d\n", addrKey, data.Seq)

	ackData := TrafficSimData{
		Sent:     data.Sent,
		Received: time.Now().UnixMilli(),
		Seq:      data.Seq,
	}
	ts.sendACK(conn, addr, ackData)

	if len(connection.ReceivedData) >= 10 {
		ts.reportToController(connection)
		connection.ReceivedData = nil
		connection.LostPackets = 0
	}
}

func (ts *TrafficSim) reportToController(connection *Connection) {
	// todo report from server end?
	fmt.Printf("Reporting to controller for %s: Received %d packets, Lost %d packets\n",
		connection.Addr.String(), len(connection.ReceivedData), connection.LostPackets)
}

func (ts *TrafficSim) monitorConnections() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ts.ConnectionsMu.Lock()
		for addrKey, conn := range ts.Connections {
			if time.Since(conn.LastResponse) > 10*time.Second {
				conn.LostPackets++
				fmt.Printf("Packet loss detected for %s\n", addrKey)
				// todo trigger alert / MTR test
			}
		}
		ts.ConnectionsMu.Unlock()
	}
}

func (ts *TrafficSim) isAgentAllowed(agentID primitive.ObjectID) bool {
	for _, allowedAgent := range ts.AllowedAgents {
		if allowedAgent == agentID {
			return true
		}
	}
	return false
}

func (ts *TrafficSim) Start(dC chan ProbeData) {
	if ts.IsServer {
		go ts.monitorConnections()
		ts.runServer()
	} else {
		ts.runClient(dC)
	}
}
