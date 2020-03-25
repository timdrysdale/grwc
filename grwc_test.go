package grwc

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/reconws"
	"github.com/timdrysdale/srgob"
)

func init() {

	log.SetLevel(log.ErrorLevel)

}

func TestExclusiveConnectionID(t *testing.T) {

	//time.Sleep(time.Millisecond)

	wsReport := make(chan reconws.WsMessage)

	// Create test server with the echo handler.
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		report(w, r, wsReport)
	}))
	defer s.Close()

	destination := "ws" + strings.TrimPrefix(s.URL, "http") //s.URL //"ws://localhost:8081"

	config := Config{
		Destination:         destination,
		ExclusiveConnection: true,
	}

	c, err := New(&config)

	if err != nil {
		t.Errorf("Problem creating client")
	}

	closed := make(chan struct{})
	defer close(closed)

	go c.Run(closed)

	time.Sleep(time.Millisecond)

	message := []byte("Foo")

	c.Send <- message

	select {
	case <-time.After(time.Millisecond):
		t.Errorf("Report timed out")
	case msg, ok := <-wsReport:
		if ok {

			//decode gob
			var srMsg srgob.Message
			rdr := bytes.NewReader(msg.Data)
			decoder := gob.NewDecoder(rdr)
			err := decoder.Decode(&srMsg)
			if err != nil {
				t.Errorf("Error decoding gob %v", err)
			}

			if bytes.Compare(srMsg.Data, message) != 0 {
				t.Errorf("Message did not match. Want: %v\nGot : %v\n", message, msg)
			}
			expectedLen := 6
			actualLen := len(srMsg.ConnectionID)
			if actualLen != expectedLen {
				t.Errorf("ConnectionID wrong length. Wanted %d\nGot : %d\n", expectedLen, actualLen)
			}
		} else {
			t.Errorf("Report channel not ok")
		}

	}

}
func TestNonExclusiveConnectionID(t *testing.T) {

	//time.Sleep(time.Millisecond)

	wsReport := make(chan reconws.WsMessage)

	// Create test server with the echo handler.
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		report(w, r, wsReport)
	}))
	defer s.Close()

	destination := "ws" + strings.TrimPrefix(s.URL, "http") //s.URL //"ws://localhost:8081"

	config := Config{
		Destination:         destination,
		ExclusiveConnection: false,
	}

	c, err := New(&config)

	if err != nil {
		t.Errorf("Problem creating client")
	}

	closed := make(chan struct{})
	defer close(closed)

	go c.Run(closed)

	time.Sleep(time.Millisecond)

	message := []byte("Foo")

	c.Send <- message

	select {
	case <-time.After(time.Millisecond):
		t.Errorf("Report timed out")
	case msg, ok := <-wsReport:
		if ok {

			//decode gob
			var srMsg srgob.Message
			rdr := bytes.NewReader(msg.Data)
			decoder := gob.NewDecoder(rdr)
			err := decoder.Decode(&srMsg)
			if err != nil {
				t.Errorf("Error decoding gob %v", err)
			}

			if bytes.Compare(srMsg.Data, message) != 0 {
				t.Errorf("Message did not match. Want: %v\nGot : %v\n", message, msg)
			}
			expectedLen := 1
			actualLen := len(srMsg.ConnectionID)
			if actualLen != expectedLen {
				t.Errorf("ConnectionID wrong length. Wanted %d\nGot : %d\n", expectedLen, actualLen)
			}
			expectedID := "*"
			if srMsg.ConnectionID != expectedID {
				t.Errorf("ConnectionID wrong. Want: %v\nGot : %v\n", expectedID, srMsg.ConnectionID)
			}
		} else {
			t.Errorf("Report channel not ok")
		}

	}

}

func TestExclusiveEcho(t *testing.T) {

	// Create test server with the echo handler.
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		echo(w, r)
	}))
	defer s.Close()

	destination := "ws" + strings.TrimPrefix(s.URL, "http") //s.URL //"ws://localhost:8081"

	config := Config{
		Destination:         destination,
		ExclusiveConnection: true,
	}

	c, err := New(&config)

	if err != nil {
		t.Errorf("Problem creating client")
	}

	closed := make(chan struct{})
	defer close(closed)

	go c.Run(closed)

	time.Sleep(time.Millisecond)

	message := []byte("Foo")

	c.Send <- message

	select {
	case <-time.After(time.Millisecond):
		t.Errorf("Receive timed out")
	case msg, ok := <-c.Receive:
		if ok {
			if bytes.Compare(msg, message) != 0 {
				t.Errorf("Message did not match. Want: %v\nGot : %v\n", message, msg)
			}
		} else {
			t.Errorf("Channel not ok")
		}

	}

}

func TestExclusiveMultipleMessages(t *testing.T) {

	// Create test server with the echo handler.
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		echo(w, r)
	}))
	defer s.Close()

	destination := "ws" + strings.TrimPrefix(s.URL, "http") //s.URL //"ws://localhost:8081"

	config := Config{
		Destination:         destination,
		ExclusiveConnection: true,
	}

	c, err := New(&config)

	if err != nil {
		t.Errorf("Problem creating client")
	}

	closed := make(chan struct{})
	defer close(closed)

	go c.Run(closed)

	time.Sleep(time.Millisecond)

	for _, char := range "ab" {

		message := []byte(string(char))
		expected := message

		c.Send <- message

		select {
		case <-time.After(time.Millisecond):
			t.Errorf("Receive timed out")
		case msg, ok := <-c.Receive:
			if ok {
				if bytes.Compare(msg, expected) != 0 {
					t.Errorf("Message did not match. Want: %v\nGot : %v\n", expected, msg)
				}
			} else {
				t.Errorf("Channel not ok")
			}

		}
	}

}

var upgrader = websocket.Upgrader{}

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		err = c.WriteMessage(mt, message)
		if err != nil {
			break
		}
	}
}

func report(w http.ResponseWriter, r *http.Request, msgChan chan reconws.WsMessage) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		msgChan <- reconws.WsMessage{Data: message, Type: mt}
	}
}

func suppressLog() {
	var ignore bytes.Buffer
	logignore := bufio.NewWriter(&ignore)
	log.SetOutput(logignore)
}

func displayLog() {
	log.SetOutput(os.Stdout)
}

//https://rosettacode.org/wiki/Generate_lower_case_ASCII_alphabet#Go
func loweralpha() string {
	p := make([]byte, 26)
	for i := range p {
		p[i] = 'a' + byte(i)
	}
	return string(p)
}
