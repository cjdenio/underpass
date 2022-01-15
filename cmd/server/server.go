package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"

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

	listeners      map[int]chan models.ClientMessage
	listenersMutex sync.RWMutex
}

var tunnels = make(map[string]*Tunnel)

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

		tunnels[subdomain] = &t

		c, err := upgrader.Upgrade(rw, r, nil)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(err.Error()))
			return
		}

		writeMutex := sync.Mutex{}

		writeMutex.Lock()
		err = util.WriteMsgPack(c, models.ServerMessage{
			Type:      "subdomain",
			Subdomain: subdomain,
		})
		if err != nil {
			log.Println(err)
		}
		writeMutex.Unlock()

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
					t.listenersMutex.Lock()
					close(t.listeners[message.RequestID])
					delete(t.listeners, message.RequestID)
					t.listenersMutex.Unlock()
					continue
				}

				t.listenersMutex.RLock()
				t.listeners[message.RequestID] <- message
				t.listenersMutex.RUnlock()
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
					writeMutex.Lock()
					err = util.WriteMsgPack(c, models.ServerMessage{
						Type:      "close",
						RequestID: req.RequestID,
					})
					if err != nil {
						log.Println(err)
					}
					writeMutex.Unlock()
				} else {
					// Infer based on data type
					switch data := req.Data.(type) {
					case *http.Request:
						marshalled := util.MarshalRequest(data)

						writeMutex.Lock()
						err = util.WriteMsgPack(c, models.ServerMessage{
							Type:      "request",
							RequestID: req.RequestID,
							Request:   marshalled,
						})
						if err != nil {
							log.Println(err)
						}
						writeMutex.Unlock()
					case []byte:
						writeMutex.Lock()
						err = util.WriteMsgPack(c, models.ServerMessage{
							Type:      "data",
							RequestID: req.RequestID,
							Data:      data,
						})
						if err != nil {
							log.Println(err)
						}
						writeMutex.Unlock()
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

			// Pipe request body
			if r.Body != nil {
				go func() {
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
				}()
			}

			messageChannel := make(chan models.ClientMessage)
			t.listenersMutex.Lock()
			t.listeners[reqID] = messageChannel
			t.listenersMutex.Unlock()

		X:
			for {
				if message, ok := <-messageChannel; ok {
					switch message.Type {
					case "proxy_error":
						rw.WriteHeader(http.StatusBadGateway)
						rw.Write([]byte("Proxy error. See your terminal for more information."))
						t.listenersMutex.Lock()
						close(messageChannel)
						delete(t.listeners, reqID)
						t.listenersMutex.Unlock()
						break X
					case "response":
						for i, v := range message.Response.Headers {
							rw.Header().Add(i, strings.Join(v, ","))
						}
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
			rw.Write([]byte(fmt.Sprintf("Tunnel %s not found.\n\nStart it with `underpass -p PORT -s haas` 😎", subdomain)))
		}
	})

	log.Println("starting...")

	http.ListenAndServe(":80", r)
}
