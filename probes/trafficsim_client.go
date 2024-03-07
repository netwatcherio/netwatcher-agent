package probes

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"github.com/quic-go/quic-go"
	log "github.com/sirupsen/logrus"
	"io"
	"time"
)

// TrafficSimClient is the type of payload that can be sent, thisAgent would be the ID of this, otherAgent is the server
func TrafficSimClient(pp *Probe, sim *TrafficSim) {
	// todo take input for channel to pipe the output to the websocket handler

	err := checkAndGenerateCertificateIfNeeded()
	if err != nil {
		log.Errorf("Failed to check and generate certificate: %v", err)
		return
	}

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"netwatcher-agent"},
	}

	// Dial the server
	conn, err := quic.DialAddr(context.Background(), pp.Config.Target[0].Target, tlsConf, nil)
	if err != nil {
		log.Errorf("Failed to dial: %v", err)
		sim.Errored = true
		return
	}
	defer func(conn quic.Connection, code quic.ApplicationErrorCode, s string) {
		err := conn.CloseWithError(code, s)
		if err != nil {
			log.Errorf("Failed to close connection: %v", err)
			sim.Errored = true
			return
		}
	}(conn, 0, "")

	sim.Conn = &conn

	stream, err := (*sim.Conn).OpenStreamSync(context.Background())
	if err != nil {
		log.Errorf("Failed to open stream: %v", err)
		sim.Errored = true
		return
	}
	defer func(stream quic.Stream) {
		err := stream.Close()
		if err != nil {
			log.Errorf("Failed to close stream: %v", err)
			sim.Errored = true
			return
		}
	}(stream)

	sim.Stream = &stream

	// to register we must send the registration payload?? fuck
	regPayload := &TrafficSimMsg{
		Type:    TrafficSimMsgType_Registration,
		Agent:   sim.OtherAgent,
		From:    sim.ThisAgent,
		Payload: "let me fucking register",
	}

	// Main loop for sending data and receiving responses
	for {
		// Replace with actual logic to construct your data packet
		if !sim.Registered {
			err = sim.sendMessage(regPayload)
			if err != nil {
				log.Errorf("Failed to send registration: %v", err)
				return
			}

			buf := make([]byte, SimMsgSize)
			n, err := io.ReadFull(*sim.Stream, buf)
			if err != nil {
				log.Errorf("something went wrong in trying to read data")
			}

			resp := new(TrafficSimMsg)

			json.Unmarshal(buf[:n], resp)
		}
		// todo handle ticker once we've actually registered??? fuck i hate low level "networking"

		// Optionally wait or handle responses
		time.Sleep(1 * time.Second) // Example sleep, adjust as needed
	}
}

// seen as we know the list of accepted agents / clients, we can presume encrypted traffic is trusted lol?
