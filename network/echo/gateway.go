package echo

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network"
	"github.com/lienkolabs/breeze/network/trusted"
)

type Stats struct {
	Token                 crypto.Token
	EstablishedConnection time.Time
	ActionCount           int
}

type connectionWithStats struct {
	conn  *trusted.SignedConnection
	start time.Time
	count int
}

type Gateway struct {
	mu      sync.Mutex
	inbound map[crypto.Token]*connectionWithStats
	Actions chan []byte
}

func (g *Gateway) Read() []byte {
	return <-g.Actions
}

func (g *Gateway) Stats() []Stats {
	g.mu.Lock()
	defer g.mu.Unlock()
	all := make([]Stats, 0)
	for token, conn := range g.inbound {
		stat := Stats{
			Token:                 token,
			EstablishedConnection: conn.start,
			ActionCount:           conn.count,
		}
		all = append(all, stat)
	}
	return all
}

func NewActionsGateway(port int, credentials crypto.PrivateKey, validate network.ValidateConnection, action chan []byte) (*Gateway, error) {
	listeners, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		return nil, err
	}

	gateway := Gateway{
		mu:      sync.Mutex{},
		inbound: make(map[crypto.Token]*connectionWithStats),
		Actions: action,
	}

	messages := make(chan trusted.Message)
	shutDown := make(chan crypto.Token) // receive connection shutdown

	go func() {
		for {
			if conn, err := listeners.Accept(); err == nil {
				trustedConn, err := trusted.PromoteConnection(conn, credentials, validate)
				if err != nil {
					conn.Close()
				} else {
					gateway.inbound[trustedConn.Token] = &connectionWithStats{
						conn:  trustedConn,
						start: time.Now(),
					}
					trustedConn.Listen(messages, shutDown)
				}
			} else {
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case token := <-shutDown:
				gateway.mu.Lock()
				delete(gateway.inbound, token)
				gateway.mu.Unlock()
			case msg := <-messages:
				gateway.Actions <- msg.Data
			}
		}
	}()

	return &gateway, nil
}
