package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"path"
	"time"
)

// originally from https://github.com/leonelquinteros/hubspot/blob/master/client.go

// ClientConfig object used for client creation
type ClientConfig struct {
	APIHost     string
	APIUsername string
	APIPassword string
	HTTPTimeout time.Duration
	DialTimeout time.Duration
	TLSTimeout  time.Duration
}

// NewClientConfig constructs a ClientConfig object with the environment variables set as default
func NewClientConfig() ClientConfig {
	apiHost := "http://localhost:3000"

	return ClientConfig{
		APIHost:     apiHost,
		HTTPTimeout: 10 * time.Second,
		DialTimeout: 5 * time.Second,
		TLSTimeout:  5 * time.Second,
	}
}

// Client object
type Client struct {
	config ClientConfig
}

// NewClient constructor
func NewClient(config ClientConfig) Client {
	return Client{
		config: config,
	}
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// Request executes any HubSpot API method using the current client configuration
func (c Client) Request(method, endpoint string, data, response interface{}) error {
	// Construct endpoint URL
	u, err := url.Parse(c.config.APIHost)
	if err != nil {
		return fmt.Errorf("url.Parse(): %v", err)
	}
	u.Path = path.Join(u.Path, endpoint)

	// User authentication
	uri := u.String()

	//todo auth
	/*if c.config.APIUsername == "" || c.config.APIPassword == "" {
		return errors.New("missing user authentication data")
	}*/

	// Init request object
	var req *http.Request

	// Send data?
	if data != nil {
		// Encode data to JSON
		dataEncoded, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("%v", err)
		}
		buf := bytes.NewBuffer(dataEncoded)

		// Create request
		req, err = http.NewRequest(method, uri, buf)
	} else {
		// Create no-data request
		req, err = http.NewRequest(method, uri, nil)
	}
	if err != nil {
		return fmt.Errorf("http.Request(): http.NewRequest(): %v", err)
	}

	// Headers
	req.Header.Add("Content-Type", "application/json")
	// todo auth??
	//req.Header.Add("Authorization", "Basic "+basicAuth(c.config.APIUsername, c.config.APIPassword))

	// Execute and read response body
	netClient := &http.Client{
		Timeout: c.config.HTTPTimeout,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: c.config.DialTimeout,
			}).Dial,
			TLSHandshakeTimeout: c.config.TLSTimeout,
		},
	}
	resp, err := netClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Get data?
	if response != nil {
		if len(body) > 0 {
			err = json.Unmarshal(body, &response)
			if err != nil {
				return fmt.Errorf("%v \n%s", err, string(body))
			}
		}
	} else {
		var prettyJSON bytes.Buffer
		error := json.Indent(&prettyJSON, body, "", "\t")
		if error != nil {
			return err
		}
		log.Printf("\n" + string(prettyJSON.Bytes()))
	}

	// Return HTTP errors
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return fmt.Errorf("API error: %d - %s \n%s", resp.StatusCode, resp.Status, string(body))
	}

	// Done!
	return nil
}
