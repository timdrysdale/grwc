package grwc

import (
	"bytes"
	"context"
	"encoding/gob"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/reconws"
	"github.com/timdrysdale/srgob"
)

// ********************************************************************************

// grwc assumes a single outgoing connection, but we still run relay goros
// because we need to send an initial message to set connectionID and destination
func New(config *Config) *Client {

	connectionID := uuid.New().String()[:3]
	if !config.ExclusiveConnection {
		connectionID = "*"
	}

	c := &Client{
		ConnectionID: connectionID,
		Topic:        config.Topic,
		Destination:  config.Destination,
		Send:         make(chan []byte),
		SendAdmin:    make(chan srgob.Message),
		Receive:      make(chan []byte),
		ReceiveAdmin: make(chan srgob.Message),
	}

	return c
}

func (c *Client) Run(ready chan struct{}) {

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
	go c.Websocket.Reconnect(c.Context, c.Destination) //TODO sanity check the destination
	//user must check stats to learn of errors - TODO, are we doing stats?
	// an RPC style return on start is of limited value because clients are long lived
	// so we'll need to check the stats later anyway; better just to do things one way

	//TODO fix the possibility of a race for first message ...
	// TODO drain send? Or just say that user must wait until Ready is closed before sending?

	// send the first message to set up the connectionID and topic.

	hello := srgob.Message{
		Topic:        c.Topic,
		ConnectionID: c.ConnectionID,
	}

	c.SendAdmin <- hello

	close(ready)
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
		case adminMsg, ok := <-c.SendAdmin:
			// gob the struct and send
			if ok {
				gobbedMsg.Reset() //reset buffer before we encode into it
				err := encoder.Encode(adminMsg)
				if err != nil {
					log.Errorf("Error gobbing admin message %v\n", err)
					return //bail out TODO bail out sensibly...
				}
				c.Websocket.Out <- reconws.WsMessage{Data: gobbedMsg.Bytes(), Type: websocket.BinaryMessage}
			}
		case msg, ok := <-c.Send:
			// assemble an srgob struct, gob it, and send
			if ok {
				msg := srgob.Message{
					//omit admin details on all but first message
					Data: msg,
				}
				gobbedMsg.Reset() //reset buffer before we encode into it
				err := encoder.Encode(msg)
				if err != nil {
					log.Errorf("Error gobbing message %v\n", err)
					return //bail out TODO bail out sensibly...
				}
				c.Websocket.Out <- reconws.WsMessage{Data: gobbedMsg.Bytes(), Type: websocket.BinaryMessage}
			}
		}
	}
}

// relay messages from websocket server to the hub until stopped
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
				if srMsg.ConnectionID != "" && srMsg.Topic != "" { //need both to count as admin message
					c.ReceiveAdmin <- srMsg
				} else { //assume data message
					c.Receive <- srMsg.Data
				}
			}
		}
	}
}
