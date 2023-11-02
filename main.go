package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/kataras/iris/v12/websocket"
	"log"
	"os"
	"os/signal"
	"runtime"
	"time"
)

const (
	endpoint              = "ws://localhost:8080/agent_ws"
	namespace             = "default"
	dialAndConnectTimeout = 5 * time.Second
)

// this can be shared with the server.go's.
// `NSConn.Conn` has the `IsClient() bool` method which can be used to
// check if that's is a client or a server-side callback.
var clientEvents = websocket.Namespaces{
	namespace: websocket.Events{
		websocket.OnNamespaceConnected: func(c *websocket.NSConn, msg websocket.Message) error {
			log.Printf("connected to namespace: %s", msg.Namespace)
			return nil
		},
		websocket.OnNamespaceDisconnect: func(c *websocket.NSConn, msg websocket.Message) error {
			log.Printf("disconnected from namespace: %s", msg.Namespace)
			return nil
		},
		"chat": func(c *websocket.NSConn, msg websocket.Message) error {
			log.Printf("%s", string(msg.Body))
			return nil
		},
	},
}

func main() {
	fmt.Printf("Starting NetWatcher Agent...\n")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		for _ = range c {
			shutdown()
			return
		}
	}()
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(dialAndConnectTimeout))
	defer cancel()

	// WebSocket server endpoint
	endpoint := "ws://localhost:8080/agent_ws"

	// Bearer token for authentication
	bearerToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpdGVtX2lkIjoiNjU0MTM1Y2M3YTMyOWE2NWVjY2Y4ODk1Iiwic2Vzc2lvbl9pZCI6IjY1NDQyM2Q4NjIxZWY3MzM1OWRmNDRhNCJ9.7rxo0I8Tid7oyQDBCMnPTBD7UXVQKCdahXXtwB9MTvY"

	// Create a custom Gobwas dialer with headers
	dialer := websocket.GobwasDialer(websocket.GobwasDialerOptions{Header: websocket.GobwasHeader{"Authorization": []string{"Bearer " + bearerToken}}})

	client, err := websocket.Dial(ctx, dialer, endpoint, clientEvents)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	cc, err := client.Connect(ctx, namespace)
	if err != nil {
		panic(err)
	}

	cc.Emit("chat", []byte("Hello from Go client side!"))

	fmt.Fprint(os.Stdout, ">> ")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		if !scanner.Scan() {
			log.Printf("ERROR: %v", scanner.Err())
			return
		}

		text := scanner.Bytes()

		if bytes.Equal(text, []byte("exit")) {
			if err := cc.Disconnect(nil); err != nil {
				log.Printf("reply from server: %v", err)
			}
			break
		}

		ok := cc.Emit("chat", text)
		if !ok {
			break
		}

		fmt.Fprint(os.Stdout, ">> ")
	}
	// try running this program twice or/and run the server's http://localhost:8080 to check the browser client as well.

	/*setup()
	clientCfg := api.ClientConfig{
		APIHost:     os.Getenv("HOST"),
		HTTPTimeout: 10 * time.Second,
		DialTimeout: 5 * time.Second,
		TLSTimeout:  5 * time.Second,
	}
	client := api.NewClient(clientCfg)

	// initialize the apiClient from api
	// todo make this a loop that checks periodically as well as handles the errors and retries
	apiClient := api.Data{
		Client: client,
	}

	apiRequest := api.ApiRequest{ID: os.Getenv("ID"), PIN: os.Getenv("PIN")}

	// init queue
	queueReq := api.ApiRequest{
		PIN:   apiRequest.PIN,
		ID:    apiRequest.ID,
		Data:  nil,
		Error: "",
	}
	checkDataCh := make(chan api.CheckData)
	agentC := make(chan []api.AgentCheck)

	workers.InitQueueWorker(checkDataCh, queueReq, clientCfg)
	workers.InitCheckWorker(agentC, checkDataCh)

	var updateReceived = false

	// todo keep track of running tests once started, tests actively running cannot be changed only removed or *disabled
	go func(cd chan []api.AgentCheck, received bool) {
		for {
			err := apiClient.Initialize(&apiRequest)
			if err != nil {
				fmt.Println(err)
			}

			b, err := json.Marshal(apiRequest.Data)
			if err != nil {
				log.Println(err)
			}
			log.Println("Config received: ", string(b))

			var ce []api.AgentCheck

			err = json.Unmarshal(b, &ce)
			if err != nil {
				log.Println(err)
			}

			if len(ce) <= 0 {
				fmt.Println("no checks received, waiting for 10 seconds")
				time.Sleep(time.Second * 10)
				continue
			}

			cd <- ce

			time.Sleep(10 * time.Second)
		}
	}(agentC, updateReceived)

	select {}*/
}

func shutdown() {
	log.Fatalf("Currently %d threads", runtime.NumGoroutine())
	log.Fatal("Shutting down NetWatcher Agent...")
}
