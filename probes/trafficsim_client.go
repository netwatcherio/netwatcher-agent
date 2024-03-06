package probes

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"github.com/quic-go/quic-go"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

func TrafficSimClient(pp *Probe) {
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
		return
	}
	defer func(conn quic.Connection, code quic.ApplicationErrorCode, s string) {
		err := conn.CloseWithError(code, s)
		if err != nil {
			log.Errorf("Failed to close connection: %v", err)
			return
		}
	}(conn, 0, "")

	stream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		log.Errorf("Failed to open stream: %v", err)
		return
	}
	defer func(stream quic.Stream) {
		err := stream.Close()
		if err != nil {
			log.Errorf("Failed to close stream: %v", err)
			return
		}
	}(stream)

	// Example of sending a registration message
	err = sendRegistration(stream, pp)
	if err != nil {
		log.Errorf("Failed to send registration: %v", err)
		return
	}

	// Main loop for sending data and receiving responses
	for {
		// Replace with actual logic to construct your data packet
		err = sendData(stream, pp)
		if err != nil {
			log.Errorf("Failed to send data: %v", err)
			return
		}

		// Optionally wait or handle responses
		time.Sleep(1 * time.Second) // Example sleep, adjust as needed
	}
}

// seen as we know the list of accepted agents / clients, we can presume encrypted traffic is trusted lol?

func sendRegistration(stream quic.Stream, pp *Probe, fromUuid primitive.ObjectID, payload string) error {
	msg := TrafficSimMsg{From: fromUuid, Agent: pp.Agent, Type: TrafficSimMsgType_Registration, Payload: payload}
	return sendMessage(stream, msg)
}

func sendData(stream quic.Stream, pp *Probe, fromUuid primitive.ObjectID, payload string) error {
	msg := TrafficSimMsg{From: fromUuid, Agent: pp.Agent, Type: TrafficSimMsgType_Payload, Payload: payload}
	return sendMessage(stream, msg)
}

func sendMessage(stream quic.Stream, msg TrafficSimMsg) error {
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = stream.Write(bytes)
	return err
}
