package models

import "net/http"

type Request struct {
	Headers http.Header `msgpack:"headers"`
	Path    string      `msgpack:"path"`
	Method  string      `msgpack:"method"`
}

type Message struct {
	Type      string  `msgpack:"type"`
	RequestID int     `msgpack:"request_id"`
	Subdomain string  `msgpack:"subdomain"`
	Request   Request `msgpack:"request"`
	Data      []byte  `msgpack:"data"`
}
