package probes

import (
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net"
	"strconv"
	"time"
)

type TrafficSim struct {
	Running       bool
	Errored       bool
	DataSend      chan string // todo change this to the trafficSim report for send/receive based on agents and such
	DataReceive   chan string // todo change this to the trafficSim report for send/receive based on agents and such
	ThisAgent     primitive.ObjectID
	OtherAgent    primitive.ObjectID
	Conn          *net.UDPConn
	IPAddress     string
	Port          int64
	IsServer      bool
	LastResponse  time.Time
	Registered    bool
	AllowedAgents []primitive.ObjectID
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
	msgBuf := make([]byte, 1024)

	ln, err := net.ListenUDP("udp4", &net.UDPAddr{Port: int(ts.Port)})
	if err != nil {
		fmt.Printf("Unable to listen on :%d\n", ts.Port)
		return
	}
	defer func(ln *net.UDPConn) {
		err := ln.Close()
		if err != nil {
			log.Error(err)
		}
	}(ln)

	fmt.Printf("Listening on :%d\n", ts.Port)

	for {
		rcvLen, addr, err := ln.ReadFromUDP(msgBuf)
		if err != nil {
			fmt.Println("Error reading from UDP:", err)
			continue
		}

		tsMsg := TrafficSimMsg{}
		err = json.Unmarshal(msgBuf[:rcvLen], &tsMsg)
		if err != nil {
			fmt.Println("Error unmarshalling message:", err)
			continue
		}

		if tsMsg.Type == TrafficSim_HELLO {
			if !ts.isAgentAllowed(tsMsg.Src) {
				fmt.Println("Ignoring message from unknown agent:", tsMsg.Src)
				continue
			}

			replyMsg, err := ts.buildMessage(TrafficSim_ACK, TrafficSimData{Received: time.Now()})
			if err != nil {
				fmt.Println("Error building reply message:", err)
				continue
			}

			_, err = ln.WriteToUDP([]byte(replyMsg), addr)
			if err != nil {
				fmt.Println("Error sending reply:", err)
			}
			ts.Conn = ln
			go ts.handleData(ln, addr)
		}
	}
}

func (ts *TrafficSim) handleData(conn *net.UDPConn, addr *net.UDPAddr) {
	for {
		msgBuf := make([]byte, 1024)
		err := conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			return
		}
		rcvLen, _, err := conn.ReadFromUDP(msgBuf)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				fmt.Println("Timeout: No response received, assuming packet loss.")
				continue
			}
			fmt.Println("Error reading from UDP:", err)
			return
		}

		tsMsg := TrafficSimMsg{}
		err = json.Unmarshal(msgBuf[:rcvLen], &tsMsg)
		if err != nil {
			fmt.Println("Error unmarshalling message:", err)
			continue
		}

		if tsMsg.Type == TrafficSim_DATA {
			data := tsMsg.Data
			seq := data.Seq
			fmt.Printf("Received data: Seq %d\n", seq)
			ts.LastResponse = time.Now()

			replyData := TrafficSimData{Received: time.Now(), Seq: seq}
			replyMsg, err := ts.buildMessage(TrafficSim_ACK, replyData)
			if err != nil {
				fmt.Println("Error building reply message:", err)
				continue
			}

			_, err = conn.WriteToUDP([]byte(replyMsg), addr)
			if err != nil {
				fmt.Println("Error sending ACK:", err)
			}
		}
	}
}

func (ts *TrafficSim) RunClient() {
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
	defer func(conn *net.UDPConn) {
		err := conn.Close()
		if err != nil {
			log.Error(err)
		}
	}(conn)

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

	err = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		log.Error(err)
		return
	}
	msgLen, fromAddr, err := conn.ReadFromUDP(msgBuf)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			fmt.Println("Timeout: No response received.")
		}
		return
	}

	// todo send over to the channel??

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
		err := conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			return
		}
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

func (ts *TrafficSim) isAgentAllowed(agentID primitive.ObjectID) bool {
	for _, allowedAgent := range ts.AllowedAgents {
		if allowedAgent == agentID {
			return true
		}
	}
	return false
}
