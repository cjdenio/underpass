package main

import (
	"flag"
	"log"
	"math/rand"
	"net/http"
	"strings"

	"github.com/cjdenio/underpass/pkg/models"
	"github.com/cjdenio/underpass/pkg/util"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type Request struct {
	RequestID int
	Close     bool
	Data      interface{}
}

type Tunnel struct {
	reqChan chan Request

	listeners map[int]chan models.ClientMessage
}

var tunnels = make(map[string]Tunnel)

var upgrader = websocket.Upgrader{}

func main() {
	host := flag.String("host", "", "")

	flag.Parse()

	r := mux.NewRouter()

	r.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("welcome to underpass"))
	}).Host(*host)

	r.HandleFunc("/start", func(rw http.ResponseWriter, r *http.Request) {
		subdomain := r.URL.Query().Get("subdomain")
		if subdomain == "" {
			subdomain, _ = gonanoid.Generate("abcdefghijklmnopqrstuvwxyz0123456789", 5)
		}

		if _, ok := tunnels[subdomain]; ok {
			// Tunnel already exists

			rw.WriteHeader(http.StatusConflict)
			rw.Write([]byte("Tunnel is already running"))
			return
		}

		reqChan := make(chan Request)
		t := Tunnel{
			reqChan:   reqChan,
			listeners: make(map[int]chan models.ClientMessage),
		}

		tunnels[subdomain] = t

		c, err := upgrader.Upgrade(rw, r, nil)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(err.Error()))
			return
		}

		err = util.WriteMsgPack(c, models.ServerMessage{
			Type:      "subdomain",
			Subdomain: subdomain,
		})
		if err != nil {
			log.Println(err)
		}

		// Listen for messages (and disconnections)
		closeChan := make(chan struct{})
		go func() {
			for {
				var message models.ClientMessage
				err = util.ReadMsgPack(c, &message)
				if err != nil {
					close(closeChan)
					c.Close()
					break
				}

				if message.Type == "close" {
					close(t.listeners[message.RequestID])
					delete(t.listeners, message.RequestID)
					continue
				}

				t.listeners[message.RequestID] <- message
			}
		}()

	X:
		for {
			select {
			case _, ok := <-closeChan:
				if !ok {
					// Clean up
					close(reqChan)
					delete(tunnels, subdomain)
					break X
				}
			case req := <-reqChan:
				if req.Close {
					// This indicates that the request body has ended
					err = util.WriteMsgPack(c, models.ServerMessage{
						Type:      "close",
						RequestID: req.RequestID,
					})
					if err != nil {
						log.Println(err)
					}
				} else {
					// Infer based on data type
					switch data := req.Data.(type) {
					case *http.Request:
						marshalled := util.MarshalRequest(data)

						err = util.WriteMsgPack(c, models.ServerMessage{
							Type:      "request",
							RequestID: req.RequestID,
							Request:   marshalled,
						})
						if err != nil {
							log.Println(err)
						}
					case []byte:
						err = util.WriteMsgPack(c, models.ServerMessage{
							Type:      "data",
							RequestID: req.RequestID,
							Data:      data,
						})
						if err != nil {
							log.Println(err)
						}
					}
				}
			}
		}
	}).Host(*host)

	r.NewRoute().HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		subdomain := strings.Split(r.Host, ".")[0]

		if t, ok := tunnels[subdomain]; ok {
			reqID := rand.Int()
			t.reqChan <- Request{
				RequestID: reqID,
				Data:      r,
			}

			if r.Body != nil {
				for {
					d := make([]byte, 50)
					n, err := r.Body.Read(d)

					if n > 0 {
						t.reqChan <- Request{
							RequestID: reqID,
							Data:      d[0:n],
						}
					}

					if err != nil {
						t.reqChan <- Request{
							RequestID: reqID,
							Close:     true,
						}
						break
					}
				}
			}

			messageChannel := make(chan models.ClientMessage)
			t.listeners[reqID] = messageChannel

		X:
			for {
				if message, ok := <-messageChannel; ok {
					switch message.Type {
					case "proxy_error":
						rw.WriteHeader(http.StatusBadGateway)
						rw.Write([]byte("Proxy error. See your terminal for more information."))
						close(messageChannel)
						delete(t.listeners, reqID)
						break X
					case "response":
						rw.WriteHeader(message.Response.StatusCode)
					case "data":
						rw.Write(message.Data)
					}
				} else {
					break
				}
			}
		} else {
			rw.WriteHeader(http.StatusNotFound)
			rw.Write([]byte("Tunnel not found."))
		}
	})

	log.Println("starting...")

	http.ListenAndServe(":80", r)
}
