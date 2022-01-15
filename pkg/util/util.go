package util

import (
	"fmt"
	"net/http"

	"github.com/cjdenio/underpass/pkg/models"
	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack/v5"
)

func MarshalRequest(r *http.Request) models.Request {
	return models.Request{
		Headers: r.Header,
		Path:    r.RequestURI,
		Method:  r.Method,
	}
}

func MarshalResponse(r *http.Response) models.Response {
	return models.Response{
		Headers:    r.Header,
		StatusCode: r.StatusCode,
	}
}

// Writes a MessagePack-encoded binary string to a WebSocket connection
func WriteMsgPack(c *websocket.Conn, v interface{}) error {
	marshalled, err := msgpack.Marshal(v)
	if err != nil {
		return err
	}

	err = c.WriteMessage(websocket.BinaryMessage, marshalled)
	if err != nil {
		return err
	}

	return nil
}

func ReadMsgPack(c *websocket.Conn, v interface{}) error {
	_, m, err := c.ReadMessage()
	if err != nil {
		return err
	}

	err = msgpack.Unmarshal(m, v)
	if err != nil {
		fmt.Println("unmarshal err")
		fmt.Println(err)
		return err
	}

	return nil
}
