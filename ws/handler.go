package ws

import (
	"context"
	"encoding/json"
	"github.com/kataras/iris/v12/websocket"
	"github.com/kataras/neffos"
	"github.com/netwatcherio/netwatcher-agent/probes"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

const (
	namespace             = "agent"
	dialAndConnectTimeout = 5 * time.Second
)

type WebSocketHandler struct {
	Host             string
	Pin              string
	ID               string
	HostWS           string
	Events           []*WebSocketEvent `json:"events"`
	connection       *websocket.NSConn
	Namespaces       *websocket.Namespaces
	RestClientConfig RestClientConfig
	ProbeGetCh       chan probes.Probe
}

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

func (wsH *WebSocketHandler) InitWS() error {
	clientCfg := RestClientConfig{
		APIHost:     wsH.Host,
		HTTPTimeout: 10 * time.Second,
		DialTimeout: 5 * time.Second,
		TLSTimeout:  5 * time.Second,
	}
	wsH.RestClientConfig = clientCfg

	wsH.connectWithRetry(nil)
	return nil
}
func (wsH *WebSocketHandler) getBearerToken() (string, error) {

	loginC := NewClient(wsH.RestClientConfig)

	loginReq := agentLogin{
		PIN: wsH.Pin,
		ID:  wsH.ID,
	}

	var agentLoginR = agentLoginResp{}

	// initialize the apiClient from api
	// todo make this a loop that checks periodically as well as handles the errors and retries
	err := loginC.Request("POST", "/agent/login", &loginReq, &agentLoginR)

	if err != nil {
		log.Error(err)
		return "", err
	}

	bearerToken := agentLoginR.Token
	return bearerToken, nil
}

func (wsH *WebSocketHandler) loadNamespaces() websocket.Namespaces {
	wsH.inboundEvents()

	nss := make(websocket.Namespaces)

	for _, ee := range wsH.Events {
		nss.On(ee.Namespace, string(ee.EventType), ee.Func)
	}

	return nss
}

func (wsH *WebSocketHandler) inboundEvents() {
	var namespace = "agent"

	wsH.Events = append(wsH.Events, &WebSocketEvent{
		Namespace: namespace,
		EventType: EventTypeWS(websocket.OnNamespaceConnected),
		Func: func(c *websocket.NSConn, msg websocket.Message) error {
			log.Printf("connected to namespace: %s", msg.Namespace)
			return nil
		}})

	wsH.Events = append(wsH.Events, &WebSocketEvent{
		Namespace: namespace,
		EventType: EventTypeWS(websocket.OnNamespaceDisconnect),
		Func: func(c *websocket.NSConn, msg websocket.Message) error {
			log.Printf("disconnected from namespace: %s", msg.Namespace)
			wsH.connectWithRetry(nil)
			//todo handle and reconnect if disconnects?
			return nil
		}})

	wsH.Events = append(wsH.Events, &WebSocketEvent{
		Namespace: namespace,
		EventType: eventTypeWS_ProbeGet,
		Func: func(nsConn *websocket.NSConn, msg websocket.Message) error {

			log.Printf("%s", string(msg.Body))
			var pp []probes.Probe
			err := json.Unmarshal(msg.Body, &pp)
			if err != nil {
				return err
			}
			log.Info("Loaded probes into memory...")
			for _, pro := range pp {
				log.Infof("Sending probe to channel for loading/processing - Type: %s, Target: %s", pro.Type, pro.Config.Target)
				wsH.ProbeGetCh <- pro
			}

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

type DataToBeSent struct {
	Data []byte
	// Add other fields if necessary
}

func (wsH *WebSocketHandler) connectWithRetry(outgoingMessages chan DataToBeSent) {
	// Define your retry strategy: initial delay, max delay, etc.
	initialDelay := 1 * time.Second
	maxDelay := 120 * time.Second
	delay := initialDelay

	for {
		token, err := wsH.getBearerToken()
		if err != nil {
			log.Errorf("Error connecting to websocket, retrying in %s: %v", delay, err)
			time.Sleep(delay)
			if delay < maxDelay {
				delay *= 2
			}
			continue
		}

		client, err := wsH.connectWS(wsH.HostWS, token)
		if err == nil {
			// Connected successfully, reset delay
			delay = initialDelay
			wsH.handleConnection(client)
			return
		}

		// Connection failed, retry with exponential backoff

		// todo, if being backed off for invalid authentication/token, have it re-authenticate to get new token
		log.Errorf("Error connecting to websocket, retrying in %s: %v", delay, err)
		time.Sleep(delay)
		if delay < maxDelay {
			delay *= 2
		}
	}
}

func (wsH *WebSocketHandler) handleConnection(client *neffos.Client) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(dialAndConnectTimeout))
	defer cancel()

	cc, err := client.Connect(ctx, namespace)
	if err != nil {
		log.Error("error connecting to websocket")
	}

	wsH.connection = cc

	// request initial data
	cc.Emit("probe_get", []byte("give me probe information"))
}

func (wsH *WebSocketHandler) connectWS(hostWS string, bearerToken string) (*neffos.Client, error) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(dialAndConnectTimeout))
	defer cancel()

	dialer := websocket.GobwasDialer(websocket.GobwasDialerOptions{Header: websocket.GobwasHeader{"Authorization": []string{"Bearer " + bearerToken}}})
	return websocket.Dial(ctx, dialer, hostWS, wsH.loadNamespaces())
}
