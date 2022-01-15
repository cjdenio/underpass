package models

import (
	"net/http"
)

type Request struct {
	Headers http.Header `msgpack:"headers"`
	Path    string      `msgpack:"path"`
	Method  string      `msgpack:"method"`
}

type Response struct {
	Headers    http.Header `msgpack:"headers"`
	StatusCode int         `msgpack:"status"`
}

// Represents a message passed from the server to the client
type ServerMessage struct {
	Type      string  `msgpack:"type"`
	RequestID int     `msgpack:"request_id"`
	Subdomain string  `msgpack:"subdomain"`
	Request   Request `msgpack:"request"`
	Data      []byte  `msgpack:"data"`
}

// Represents a message passed from the client to the server
type ClientMessage struct {
	Type      string   `msgpack:"type"`
	RequestID int      `msgpack:"request_id"`
	Response  Response `msgpack:"response"`
	Data      []byte   `msgpack:"data"`
}
