package tunnel

import (
	"fmt"

	"github.com/cjdenio/underpass/pkg/client/models"
	"github.com/fatih/color"
	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack/v5"
)

type Tunnel struct {
	Subdomain string
	closeChan chan error
}

func Connect(url string) (*Tunnel, error) {
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}

	subdomainChan := make(chan string)
	closeChan := make(chan error)

	go func() {
		for {
			var msg models.Message

			_, m, err := c.ReadMessage()
			if err != nil {
				closeChan <- err
				close(closeChan)
				break
			}
			msgpack.Unmarshal(m, &msg)

			switch msg.Type {
			case "subdomain":
				subdomainChan <- msg.Subdomain
				close(subdomainChan)
			case "request":
				color.New(color.FgHiBlack).Print("--> ")
				fmt.Printf("%s %s\n", msg.Request.Method, msg.Request.Path)
			}
		}
	}()

	subdomain := <-subdomainChan

	return &Tunnel{
		Subdomain: subdomain,
		closeChan: closeChan,
	}, nil
}

func (t *Tunnel) Wait() error {
	return <-t.closeChan
}
