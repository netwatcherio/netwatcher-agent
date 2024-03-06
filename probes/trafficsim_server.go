package probes

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/quic-go/quic-go"
	log "github.com/sirupsen/logrus"
)

func TrafficSimServer(pp *Probe) {
	err := checkAndGenerateCertificateIfNeeded()
	if err != nil {
		log.Errorf("Failed to check and generate certificate: %v", err)
		return
	}

	listener, err := quic.ListenAddr(pp.Config.Target[0].Target, generateTLSConfig(), nil)
	if err != nil {
		log.Errorf("Failed to listen: %v", err)
		return
	}
	defer func(listener *quic.Listener) {
		err := listener.Close()
		if err != nil {
			log.Errorf("Failed to close listener: %v", err)
			return
		}
	}(listener)

	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			log.Errorf("Failed to accept connection: %v", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn quic.Connection) {
	for {
		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			log.Errorf("Failed to accept stream: %v", err)
			break
		}

		go handleStream(stream)
	}
}

func handleStream(stream quic.Stream) {
	defer func(stream quic.Stream) {
		err := stream.Close()
		if err != nil {
			log.Errorf("Failed to close stream: %v", err)
		}
	}(stream)

	var buf [1024]byte
	n, err := stream.Read(buf[:])
	if err != nil {
		log.Printf("Failed to read from stream: %v", err)
		return
	}

	var msg map[string]interface{}
	if err := json.Unmarshal(buf[:n], &msg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return
	}

	processMessage(stream, msg)
}

func processMessage(stream quic.Stream, msg map[string]interface{}) {
	switch msg["type"] {
	case "registration":
		fmt.Println("Handling registration")
		// handle registration
	case "data":
		fmt.Println("Handling data")
		// handle data
	default:
		fmt.Printf("Unknown message type: %v\n", msg["type"])
	}
}
