package tunnel

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"context"

	"github.com/cjdenio/underpass/pkg/client/models"
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

	go func() {
		for {
			var msg models.Message

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
				color.New(color.FgHiBlack).Printf("%d --> ", msg.RequestID)
				fmt.Printf("%s %s\n", msg.Request.Method, msg.Request.Path)

				read, write := io.Pipe()
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				request, _ := http.NewRequestWithContext(ctx, msg.Request.Method, address+msg.Request.Path, read)
				t.activeRequests[msg.RequestID] = write

				// Perform the request in a goroutine
				go func(request *http.Request, reqId int, cancel context.CancelFunc) {
					defer cancel()
					_, err = http.DefaultClient.Do(request)
					if err != nil {
						color.New(color.FgRed).Printf("%d --> ", msg.RequestID)
						fmt.Printf("Proxy error: %s\n", err)
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
			default:
				fmt.Printf("unknown type: %s\n", msg.Type)
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
