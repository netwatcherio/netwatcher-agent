package probes

import (
	"fmt"
	"net"
	"strings"
)

// Assuming you have a function to get the server's unique UUID
func getServerUUID() string {
	return "server-agent-UUID" // This should be securely stored and consistently used
}

// Assuming a function that checks if the UUID is trusted by comparing the bcrypted version
func isUUIDTrusted(incomingUUID, serverUUID string) bool {
	// This would involve comparing the bcrypted incoming UUID with the list of trusted UUIDs
	// For simplicity, this function always returns false here, replace with your actual logic
	return false
}

func RunServer(pp *Probe) {
	addr, _ := net.ResolveUDPAddr("udp", pp.Config.Target[0].Target)
	conn, _ := net.ListenUDP("udp", addr)
	defer conn.Close()

	clients := make(map[string]string) // Map of client UUID to IP address for tracking

	for {
		buffer := make([]byte, 1024)
		bytesRead, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error on reading from UDP:", err)
			continue
		}

		incoming := string(buffer[:bytesRead])
		fmt.Println("[INCOMING]", incoming)

		// Assuming the incoming message is "register:<UUID>"
		parts := strings.Split(incoming, ":")
		if len(parts) != 2 {
			fmt.Println("Invalid message format")
			continue
		}

		command, incomingUUID := parts[0], parts[1]

		// Handle re-registration for IP changes
		if command == "register" {
			serverUUID := getServerUUID()
			if isUUIDTrusted(incomingUUID, serverUUID) {
				currentIP := clients[incomingUUID]
				if currentIP != "" && currentIP != remoteAddr.String() {
					// IP has changed, inform client to re-register
					fmt.Printf("IP change detected for UUID %s. Previous IP: %s, New IP: %s\n", incomingUUID, currentIP, remoteAddr.String())
					resp := "re-register"
					conn.WriteToUDP([]byte(resp), remoteAddr)
					// Optionally, you can immediately update the IP, or wait for the client to re-register
					// clients[incomingUUID] = remoteAddr.String()
				} else {
					// New client or no IP change, add/update client
					clients[incomingUUID] = remoteAddr.String()
					resp := "Registered"
					conn.WriteToUDP([]byte(resp), remoteAddr)
					fmt.Printf("[INFO] Responded to %s with %s\n", incomingUUID, resp)
				}
			} else {
				fmt.Println("Untrusted or invalid UUID:", incomingUUID)
			}
		}
		// Additional commands can be handled here
	}
}
