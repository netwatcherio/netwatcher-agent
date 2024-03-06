```
 Client receives list of probes + traffic simulation destination endpoints (calculated by their public IP, and defined port)
Server also receives list of calculated endpoints with their agent UUID + general information for comparision

When client initiates handshake, it already knows the destination UUID of the target agent for the traffic simulation using UDP hole punching
Both ends know each party, so when it starts to send it's simulated traffic that contains:
    - Timestamp, public IP?, and maybe any other information we see fit, with roughly <32kbits
The server will then validate that the first section of the data contains the bcrypted version of the allowd UUID using it's own uuid as the salt to be able to compare
This way it will prevent abuse, if it is not an allowed source/validated target, refuse connection and drop packets. 



When displaying the traffic server based probes, we need to link back to the other ones because if a client connects to the server
We need to also run the tests back to it so there is bidirectional communication with it, and the server version will also log it's tests to the client at the same time


Agent Data Format:
Server will start UDP listener, and wait for connections
1. When the server receives a connection request, and it is the first time, it will include the following format
    -> We will include the brypted client UUID, salted with the server agents's UUID
    FROM_UUID=%=TO_UUID=%===register==BCRYPT_SENDER_UUID
2. Once the client is registered, the server will respond back with:
    -> We will include the brypted client UUID, salted with the server agents's UUID
    TO_UUID=%=FROM_UUID=%===registered==BCRYPT_SENDER_UUID
3. Once the server responds with OK to let the client know it's acknowledged it, it will start to send latency/ping data for metric collection
    FROM_UUID=%=TO_UUID=%===OK
4. The client and both the server now have an established connection
    -> When client is sending data to the server it will look like:
        SERVER_UUID=%=CLIENT_UUID=%===SEND_TIMESTAMP==SEQUENCE_NUMBER
    -> When the server is responding to the client, it will respond with SERVER_UUID=%=CLIENT_UUID=%===RECEIVE_TIMESTAMP==SEQUENCE_NUMBER
        -> If the client does not receive a response from the server within an expected time (750ms?), we can count that packet as lost
        -> We need to continue regardless and keep sending packets at the 500-1000ms interval, and simply check to see which packets we missed
        based on UUID tracking of the packet.
    -> We need to use sequence numbers to keep track of out of order packets, or if we already received them and keep track of duplicate packets
    -> The server will also attempt to send it's own data to the client so we can measure the statistics in reverse, if we don't receive one, 
    can we do re-transmits or should be mark as lost?
        CLIENT_UUID=%=SERVER_UUID=%===SEND_TIMESTAMP==SEQUENCE_NUMBER
    -> We can use the format of destination first for both to determine the direction because we know both ends already, so it should be easy to figure out
    -> When the client responds to the server's latency check, it will also do the same but in the opposite direction as previously described.

We also need to include the original probe ID in the handshake so the server can still report it's findings to the backend and have it link to the original probe ID, yet include the agent/client destination as the target, and the ID of the server somewhere, or maybe we can infer it from the probe's original target to begin with, eg. no extra information in target field, we can assume it was to the server, vs if it has the other agent ID it was to the agent instead of the server?

When both ends are connected and running their tests, we will also need to keep track of latency for each packet/packets, sequence and out of order / duplicate packets.
```
