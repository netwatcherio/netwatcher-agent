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
	Running       bool
	Errored       bool
	DataSend      chan string
	DataReceive   chan string
	ThisAgent     primitive.ObjectID
	OtherAgent    primitive.ObjectID
	Conn          *net.UDPConn
	IPAddress     string
	Port          int64
	IsServer      bool
	LastResponse  time.Time
	Registered    bool
	AllowedAgents []primitive.ObjectID
	Connections   map[string]*Connection
	ConnectionsMu sync.RWMutex
}

type Connection struct {
	Addr         *net.UDPAddr
	LastResponse time.Time
	LostPackets  int
	ReceivedData []TrafficSimData
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
	Sent     time.Time `json:"sent"`
	Received time.Time `json:"received"`
	Seq      int       `json:"seq"`
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

func (ts *TrafficSim) RunServer() {
	ln, err := net.ListenUDP("udp4", &net.UDPAddr{Port: int(ts.Port)})
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
		ts.sendACK(conn, addr, TrafficSimData{Sent: time.Now()})
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

	ts.sendACK(conn, addr, TrafficSimData{Received: time.Now(), Seq: data.Seq})

	if len(connection.ReceivedData) >= 10 {
		ts.reportToController(connection)
		connection.ReceivedData = nil
		connection.LostPackets = 0
	}
}

func (ts *TrafficSim) reportToController(connection *Connection) {
	// Implement your reporting logic here
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

func (ts *TrafficSim) Start() {
	if ts.IsServer {
		go ts.monitorConnections()
		ts.RunServer()
	} else {
		ts.runClient()
	}
}

func (ts *TrafficSim) runClient() {
	msgBuf := make([]byte, 1024)

	toAddr, err := net.ResolveUDPAddr("udp4", ts.IPAddress+":"+strconv.Itoa(int(ts.Port)))
	if err != nil {
		fmt.Printf("Could not resolve %s:%d\n", ts.IPAddress, ts.Port)
		return
	}

	fmt.Printf("Trying to punch a hole to %s:%d\n", ts.IPAddress, ts.Port)

	conn, err := net.DialUDP("udp4", nil, toAddr)
	if err != nil {
		fmt.Printf("Unable to connect to %s:%d\n", ts.IPAddress, ts.Port)
		return
	}
	defer conn.Close()

	ts.Conn = conn

	helloMsg, err := ts.buildMessage(TrafficSim_HELLO, TrafficSimData{Sent: time.Now()})
	if err != nil {
		fmt.Println("Error building hello message:", err)
		return
	}

	_, err = conn.Write([]byte(helloMsg))
	if err != nil {
		fmt.Println("Error sending hello message:", err)
		return
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	msgLen, fromAddr, err := conn.ReadFromUDP(msgBuf)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			fmt.Println("Timeout: No response received.")
		} else {
			fmt.Println("Error reading UDP response:", err)
		}
		return
	}

	fmt.Printf("Received a UDP packet back from %s:%d\n\tResponse: %s\n",
		fromAddr.IP, fromAddr.Port, string(msgBuf[:msgLen]))

	fmt.Println("Success: NAT traversed! ^-^")

	go ts.sendDataLoop(conn, toAddr)
	go ts.receiveDataLoop(conn)
}

func (ts *TrafficSim) sendDataLoop(conn *net.UDPConn, toAddr *net.UDPAddr) {
	seq := 0
	for {
		time.Sleep(1 * time.Second) // Send data every second
		seq++
		data := TrafficSimData{Sent: time.Now(), Seq: seq}
		dataMsg, err := ts.buildMessage(TrafficSim_DATA, data)
		if err != nil {
			fmt.Println("Error building data message:", err)
			continue
		}

		_, err = conn.Write([]byte(dataMsg))
		if err != nil {
			fmt.Println("Error sending data message:", err)
		}
	}
}

func (ts *TrafficSim) receiveDataLoop(conn *net.UDPConn) {
	for {
		msgBuf := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		msgLen, _, err := conn.ReadFromUDP(msgBuf)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				fmt.Println("Timeout: No response received.")
				continue
			}
			fmt.Println("Error reading from UDP:", err)
			return
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
			fmt.Printf("Received ACK: Seq %d\n", seq)
			ts.LastResponse = time.Now()
		}
	}
}
