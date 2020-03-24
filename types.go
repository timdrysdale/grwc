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
}

type Client struct {
	Cancel              context.CancelFunc
	Context             context.Context
	ConnectionID        string
	Destination         string
	ExclusiveConnection bool
	Receive             chan []byte
	ReceiveGob          chan srgob.Message
	Send                chan []byte
	Topic               string
	Websocket           *reconws.ReconWs
}
