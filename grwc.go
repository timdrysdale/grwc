package grwc

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"net/url"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/reconws"
	"github.com/timdrysdale/srgob"
)

// ********************************************************************************

// TODO - fix the destinations /topics thing - in this implementation, destination must have
// routing for the topic, so not needed separately in the struct
// TODO - how does SA keep track of connectionID - should it just send on every message?
//        that simplifies the need for an admin channel ... is taken care of by websocket routing

// grwc assumes a single outgoing connection, but we still run relay goros
// because we need to send an initial message to set connectionID and destination
func New(config *Config) (*Client, error) {

	connectionID := uuid.New().String()[:3]
	if !config.ExclusiveConnection {
		connectionID = "*"
	}

	//early warning; reconws will sanity check the Destination too
	if !destinationOK(config.Destination) {
		return nil, errors.New("Bad Destination")
	}

	c := &Client{
		ConnectionID:        connectionID, //not in config, we just generated it
		Destination:         config.Destination,
		ExclusiveConnection: config.ExclusiveConnection,
		Send:                make(chan []byte),
		Receive:             make(chan []byte),
		ReceiveGob:          make(chan srgob.Message),
	}

	return c, nil
}

func (c *Client) Run() {

	//on exit, close client
	defer func() {
		//may panic if a client is closed just before exiting
		//but if exiting, a panic is less of an issue
		c.Cancel()
	}()

	// start the reconws
	c.Websocket = reconws.New()
	c.Context, c.Cancel = context.WithCancel(context.Background())
	go c.RelayIn()
	go c.RelayOut()
	go c.Websocket.Reconnect(c.Context, c.Destination)
	//user must check stats to learn of errors - TODO, are we doing stats?
	// an RPC style return on start is of limited value because clients are long lived
	// so we'll need to check the stats later anyway; better just to do things one way

	// caller to issue c.Cancel() to stop RelayIn() & RelayOut()
}

// relay messages from the hub to the websocket client until stopped
func (c *Client) RelayOut() {

	var gobbedMsg bytes.Buffer

	encoder := gob.NewEncoder(&gobbedMsg)

LOOP:
	for {
		select {
		case <-c.Context.Done():
			break LOOP
		case msg, ok := <-c.Send:
			// assemble an srgob struct, gob it, and send
			if ok {
				msg := srgob.Message{
					ConnectionID: c.ConnectionID,
					Data:         msg,
				}
				gobbedMsg.Reset() //reset buffer before we encode into it
				err := encoder.Encode(msg)
				if err != nil {
					log.Errorf("Error gobbing message %v\n", err)
					c.Cancel() //TODO bail out sensibly...
				}
				c.Websocket.Out <- reconws.WsMessage{Data: gobbedMsg.Bytes(), Type: websocket.BinaryMessage}
			}
		}
	}
}

// receive messages from websocket server until stopped
func (c *Client) RelayIn() {

	var srMsg srgob.Message

LOOP:
	for {
		select {
		case <-c.Context.Done():
			break LOOP
		case msg, ok := <-c.Websocket.In:
			if ok {
				//Apparently overhead of new decoder per message is not high, Rob Pike secondhand via
				//https://www.reddit.com/r/golang/comments/7ospor/gob_encoding_how_do_you_use_it_in_production/
				r := bytes.NewReader(msg.Data)
				decoder := gob.NewDecoder(r)
				err := decoder.Decode(&srMsg)
				if err != nil {
					log.Errorf("Error decoding message %v\n", err)
					return //bail out TODO bail out sensibly...
				}
				if c.ExclusiveConnection { //just receive data
					c.Receive <- srMsg.Data
				} else { //non-exclusive, need connectionID so send gob
					c.ReceiveGob <- srMsg

				}
			}
		}
	}
}

func destinationOK(urlStr string) bool {

	if urlStr == "" {
		log.Error("Can't dial an empty Url")
		return false
	}

	// parse to check, dial with original string
	u, err := url.Parse(urlStr)

	if err != nil {
		log.Error("Url:", err)
		return false
	}

	if u.Scheme != "ws" && u.Scheme != "wss" {
		log.Error("Url needs to start with ws or wss")
		return false
	}

	if u.User != nil {
		log.Error("Url can't contain user name and password")
		return false
	}

	return true
}
