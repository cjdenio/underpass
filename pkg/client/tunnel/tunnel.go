package tunnel

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"context"

	"github.com/cjdenio/underpass/pkg/models"
	"github.com/cjdenio/underpass/pkg/util"
	"github.com/fatih/color"
	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack/v5"
)

type Tunnel struct {
	Subdomain string
	Address   string

	closeChan chan error

	activeRequests map[int]*io.PipeWriter
}

func Connect(url, address string) (*Tunnel, error) {
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}

	subdomainChan := make(chan string)
	closeChan := make(chan error)

	t := &Tunnel{
		closeChan:      closeChan,
		activeRequests: make(map[int]*io.PipeWriter),
	}

	writeMutex := sync.Mutex{}

	go func() {
		for {
			var msg models.ServerMessage

			_, m, err := c.ReadMessage()
			if err != nil {
				closeChan <- err
				close(closeChan)
				break
			}
			err = msgpack.Unmarshal(m, &msg)
			if err != nil {
				fmt.Println(err)
			}

			switch msg.Type {
			case "subdomain":
				subdomainChan <- msg.Subdomain
				close(subdomainChan)
			case "request":
				read, write := io.Pipe()
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				request, _ := http.NewRequestWithContext(ctx, msg.Request.Method, address+msg.Request.Path, read)
				t.activeRequests[msg.RequestID] = write

				request.Header = msg.Request.Headers

				// Perform the request in a goroutine
				go func(request *http.Request, reqID int, cancel context.CancelFunc) {
					defer cancel()

					resp, err := http.DefaultClient.Do(request)
					if err != nil {
						color.New(color.FgHiBlack).Printf("%d --> ", msg.RequestID)
						fmt.Printf("%s %s", msg.Request.Method, msg.Request.Path)
						color.New(color.FgHiBlack).Print(" --> ")
						color.New(color.FgRed).Printf("Proxy error: %s\n", err)

						writeMutex.Lock()
						err = util.WriteMsgPack(c, models.ClientMessage{Type: "proxy_error", RequestID: reqID})
						if err != nil {
							fmt.Println(err)
						}
						writeMutex.Unlock()
						return
					}

					color.New(color.FgHiBlack).Printf("%d --> ", msg.RequestID)
					fmt.Printf("%s %s", msg.Request.Method, msg.Request.Path)
					color.New(color.FgHiBlack).Print(" --> ")
					fmt.Printf("%s\n", resp.Status)

					writeMutex.Lock()
					err = util.WriteMsgPack(c, models.ClientMessage{
						Type:      "response",
						RequestID: reqID,
						Response:  util.MarshalResponse(resp),
					})
					if err != nil {
						fmt.Println(err)
					}
					writeMutex.Unlock()

					// Read the body
					for {
						d := make([]byte, 50)
						n, err := resp.Body.Read(d)

						if n > 0 {
							writeMutex.Lock()
							util.WriteMsgPack(c, models.ClientMessage{
								Type:      "data",
								RequestID: reqID,
								Data:      d[0:n],
							})
							writeMutex.Unlock()
						}

						if err != nil {
							writeMutex.Lock()
							err = util.WriteMsgPack(c, models.ClientMessage{Type: "close", RequestID: reqID})
							if err != nil {
								fmt.Println(err)
							}
							writeMutex.Unlock()
							break
						}
					}
				}(request, msg.RequestID, cancel)
			case "close":
				if v, ok := t.activeRequests[msg.RequestID]; ok {
					v.Close()
					delete(t.activeRequests, msg.RequestID)
				}
			case "data":
				if v, ok := t.activeRequests[msg.RequestID]; ok {
					v.Write(msg.Data)
				}
			}
		}
	}()

	select {
	case subdomain := <-subdomainChan:
		t.Subdomain = subdomain
		return t, nil
	case err = <-closeChan:
		return nil, err
	}
}

func (t *Tunnel) Wait() error {
	return <-t.closeChan
}
