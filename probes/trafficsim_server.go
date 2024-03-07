package probes

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/quic-go/quic-go"
	log "github.com/sirupsen/logrus"
)

func TrafficSimServer(pp *Probe, sim *TrafficSim) {
	err := checkAndGenerateCertificateIfNeeded()
	if err != nil {
		log.Errorf("Failed to check and generate certificate: %v", err)
		sim.Errored = true
		return
	}

	listener, err := quic.ListenAddr(pp.Config.Target[0].Target, generateTLSConfig(), nil)
	if err != nil {
		log.Errorf("Failed to listen: %v", err)
		sim.Errored = true
		return
	}
	defer func(listener *quic.Listener) {
		err := listener.Close()
		if err != nil {
			log.Errorf("Failed to close listener: %v", err)
			sim.Errored = true
			return
		}
	}(listener)

	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			log.Errorf("Failed to accept connection: %v", err)
			sim.Errored = true
			continue
		}

		sim.Conn = &conn

		go sim.handleConnection()
	}
}

// handleConnection handles the incoming connection for the simulation server
func (sim *TrafficSim) handleConnection() {
	for {
		stream, err := (*sim.Conn).AcceptStream(context.Background())
		if err != nil {
			log.Errorf("Failed to accept stream: %v", err)
			break
		}

		sim.Stream = &stream

		go sim.handleStream()
	}
}

func (sim *TrafficSim) handleStream() {
	defer func(stream quic.Stream) {
		err := stream.Close()
		if err != nil {
			log.Errorf("Failed to close stream: %v", err)
		}
	}(*sim.Stream)

	var buf [SimMsgSize]byte
	n, err := (*sim.Stream).Read(buf[:])
	if err != nil {
		log.Printf("Failed to read from stream: %v", err)
		return
	}

	var msg TrafficSimMsg
	if err := json.Unmarshal(buf[:n], &msg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return
	}

	sim.processMessage(&msg)
}

func (sim *TrafficSim) processMessage(msg *TrafficSimMsg) {
	// todo send this over to the channel

	switch msg.Type {
	case TrafficSimMsgType_Registration:
		log.Warningf("handling registration from client %v", sim.ThisAgent)
		// todo handle registration
		// when registering we need to validate that we infact have the far end agent in our list of approved agents??
		// this seems less likely to be abused, and if someone does, damn they are smart...
		// mind you it is open source 🤪
		log.Warningf("registering client %v", msg)

		// todo validate from and destination agent

		log.Warningf("sending registration response to client %v", msg.From)

		msg.Payload = "registered"
		msg.From = sim.ThisAgent

		log.Info("sending information to the local ip of the client %v", (*sim.Conn).LocalAddr())
		log.Info("sending information to the remove ip of the client %v", (*sim.Conn).RemoteAddr())

		marshal, err := json.Marshal(msg)
		if err != nil {
			log.Errorf("Failed to marshal message: %v", err)
			return
		}
		(*sim.Stream).Write(marshal)
	case TrafficSimMsgType_Payload:
		fmt.Println("Handling data")
		// todo handle data
	default:
		dataRecv, err := json.Marshal(msg)
		if err != nil {
			log.Errorf("Failed to marshal message: %v", err)
			return
		}

		fmt.Printf("Unknown message type: %v\n", dataRecv)
	}
}
