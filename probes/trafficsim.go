package probes

/*

Server:
	- Listen for traffic sim data / clients on provided probe port for TrafficSim Server
	- When an agent client connects, add it to the list of clients
	- When the agent client sends data, it will be sending it's ID and the data encrypted with it's ID as the preshared key
	- If the data infact matches the allowed agents, then the data will be sent to the TrafficSim Server
	- When receiving the data, it will need to be sent over to a channel processing that data that includes the connection pointer
	  to the connected server
	- If an agent looses connection, it will need to be re-established, to do this, we should update the existing client in the list
	- When receiving payloads from agents, we will be responding to them with the same data they sent, but encrypted with the preshared key, but with the received timestamp
	-


*/
