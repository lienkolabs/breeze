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

// stats for all live connections to the gateway
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

// a gateway provides connectivity to submit actions to the breeze network
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

// port is the listening port for upcomming connections.
// credentials to provide trusted connectivity on behalf of the responsible
// for the gateway.
// validate the tokens of the new connections. use trusted.AcceptAllConnection
// for no policy
// action is the channel through which all actions received will be transmitted.
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
					fmt.Println("connected")
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
				fmt.Println("received")
				gateway.Actions <- msg.Data
			}
		}
	}()

	return &gateway, nil
}
