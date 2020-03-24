package grwc

import (
	"context"

	"github.com/timdrysdale/reconws"
	"github.com/timdrysdale/srgob"
)

type Config struct {
	Context             context.Context
	Destination         string
	ExclusiveConnection bool
	Topic               string
}

type Client struct {
	ConnectionID string
	Topic        string
	Destination  string
	Send         chan []byte
	SendAdmin    chan srgob.Message
	Receive      chan []byte
	ReceiveAdmin chan srgob.Message
	Context      context.Context
	Cancel       context.CancelFunc
	Websocket    *reconws.ReconWs
}

/*
type Rule struct {
	Id          string `json:"id"`
	Stream      string `json:"stream"`
	Destination string `json:"destination"`
}

type Client struct {
	Hub       *Hub //can access messaging hub via <client>.Hub.Messages
	Messages  *hub.Client
	Context   context.Context
	Cancel    context.CancelFunc
	Websocket *reconws.ReconWs
}
*/
