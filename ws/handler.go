package ws

import (
	"context"
	"github.com/kataras/iris/v12/websocket"
	"github.com/netwatcherio/netwatcher-agent/api"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

const (
	namespace             = "agent"
	dialAndConnectTimeout = 5 * time.Second
)

type EventTypeWS string

type WebSocketEvent struct {
	Namespace string      `json:"namespace"`
	EventType EventTypeWS `json:"event_name"`
	Func      func(nsConn *websocket.NSConn, msg websocket.Message) error
}

const (
	eventTypeWS_ProbeGet  = "probe_get"
	eventTypeWS_ProbeData = "probe_data"
	eventTypeWS_AgentGet  = "agent_get"
)

type WebSocketHandler struct {
	Events     []*WebSocketEvent `json:"events"`
	connection *websocket.NSConn
}

func (wsH *WebSocketHandler) buildEventsWS() {
	wsH.Events = append(wsH.Events, &WebSocketEvent{
		Namespace: namespace,
		EventType: eventTypeWS_ProbeGet,
		Func: func(nsConn *websocket.NSConn, msg websocket.Message) error {
			log.Info("pee pee poo poo")
			return nil
		},
	})
}

type agentLogin struct {
	PIN string `json:"pin"`
	ID  string `json:"id"`
}

type agentLoginResp struct {
	Token string `json:"token"`
	Data  Agent  `json:"data"`
}

type Agent struct {
	ID          primitive.ObjectID `bson:"_id, omitempty"json:"id"`       // id
	Name        string             `bson:"name"json:"name"form:"name"`    // name of the agentprobe
	Site        primitive.ObjectID `bson:"site"json:"site"`               // _id of mongo object
	Pin         string             `bson:"pin"json:"pin"`                 // used for registration & authentication
	Initialized bool               `bson:"initialized"json:"initialized"` // will this be used or will we use the sessions/jwt tokens?
	Location    float64            `bson:"location"json:"location"`       // logical/physical location
	CreatedAt   time.Time          `bson:"createdAt"json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt"json:"updatedAt"`
	// pin will be used for "auth" as the password, the ID will stay the same
}

func (wsH *WebSocketHandler) InitWS(host string, hostWS string, pin string, id string) {

	wsH.buildEventsWS()

	clientCfg := api.ClientConfig{
		APIHost:     host,
		HTTPTimeout: 10 * time.Second,
		DialTimeout: 5 * time.Second,
		TLSTimeout:  5 * time.Second,
	}
	loginC := api.NewClient(clientCfg)

	loginReq := agentLogin{
		PIN: pin,
		ID:  id,
	}

	var agentLoginR = agentLoginResp{}

	// initialize the apiClient from api
	// todo make this a loop that checks periodically as well as handles the errors and retries
	err := loginC.Request("POST", "/agent/login", &loginReq, &agentLoginR)

	if err != nil {
		log.Error(err)
	}

	// todo login and get bearerToken
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(dialAndConnectTimeout))
	defer cancel()

	// WebSocket server endpoint
	// Bearer token for authentication
	bearerToken := agentLoginR.Token
	// Create a custom Gobwas dialer with headers
	dialer := websocket.GobwasDialer(websocket.GobwasDialerOptions{Header: websocket.GobwasHeader{"Authorization": []string{"Bearer " + bearerToken}}})

	client, err := websocket.Dial(ctx, dialer, hostWS, clientEvents)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	cc, err := client.Connect(ctx, namespace)
	if err != nil {
		log.Error("error connecting to websocket")
	}

	wsH.connection = cc

	// todo handle channel for sending data to/receiving from websocket backend??

	cc.Emit("probe_get", []byte("give me probe information"))

	select {}
}

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
			// todo handle disconnect logic
			return nil
		},
		"probe_get": func(c *websocket.NSConn, msg websocket.Message) error {
			log.Printf("%s", string(msg.Body))
			return nil
		},
		// todo handle probe_config events and such used for pulling and parsing probe data and such
		// the ability to actively update the configuration of the agent
	},
}
