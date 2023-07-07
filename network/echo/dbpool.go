package echo

import (
	"fmt"
	"net"
	"sync"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network"
	"github.com/lienkolabs/breeze/network/trusted"
)

// DBPool is a primitive connection to offer the service of token indexation of
// messages.
type DBPool struct {
	mu         sync.Mutex
	conn       map[crypto.Token]*trusted.SignedConnection   // connection token -> conn
	indexation map[crypto.Token][]*trusted.SignedConnection // index token -> conn array
}

type DBPoolConfig struct {
	Credentials crypto.PrivateKey
	Validator   network.ValidateConnection
	ServePort   int
	Job         chan *NewIndexJob
}

type NewIndexJob struct {
	Connection *trusted.SignedConnection
	Tokens     []crypto.Token
	FromEpoch  uint64
}

func NewDBPool(config *DBPoolConfig) (*DBPool, error) {
	pool := &DBPool{
		mu:   sync.Mutex{},
		conn: make(map[crypto.Token]*trusted.SignedConnection),
	}

	listeners, err := net.Listen("tcp", fmt.Sprintf(":%v", config.ServePort))
	if err != nil {
		return nil, err
	}

	messages := make(chan trusted.Message)
	shutdown := make(chan crypto.Token)

	// accept new connecttions
	go func() {
		for {
			if conn, err := listeners.Accept(); err == nil {
				tConn, err := trusted.PromoteConnection(conn, config.Credentials, config.Validator)
				if err != nil {
					conn.Close()
				} else {
					pool.mu.Lock()
					pool.conn[tConn.Token] = tConn
					pool.mu.Unlock()
					tConn.Listen(messages, shutdown)
				}
			} else {
				return
			}
		}
	}()

	// close broken connections and process subscribe instructions
	go func() {
		for {
			select {
			case token := <-shutdown:
				pool.mu.Lock()
				if listener, ok := pool.conn[token]; ok {
					listener.Shutdown()
				}
				delete(pool.conn, token)
				pool.mu.Unlock()
			case msg := <-messages:
				if receive := ParseReceiveTokens(msg.Data); receive != nil {
					if conn, ok := pool.conn[msg.Token]; ok {
						job := NewIndexJob{
							Connection: conn,
							Tokens:     receive.Tokens,
							FromEpoch:  receive.FromEpoch,
						}
						config.Job <- &job
					}

				}
			}
		}
	}()
	return pool, nil
}

func (pool *DBPool) Index(receive *ReceiveTokens, conn *trusted.SignedConnection) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	if receive == nil {
		return
	}
	for _, token := range receive.Tokens {
		if registeredConnections, ok := pool.indexation[token]; ok {
			isNew := true
			for _, connection := range registeredConnections {
				if connection == conn {
					isNew = false
					break
				}
			}
			if isNew {
				pool.indexation[token] = append(registeredConnections, conn)
			}
		} else {
			pool.indexation[token] = []*trusted.SignedConnection{conn}
		}
	}
}

func (pool *DBPool) Shutdown() {
	pool.mu.Lock()
	for _, conn := range pool.conn {
		conn.Shutdown()
	}
	defer pool.mu.Unlock()
}

func (pool *DBPool) Broadcast(data []byte, tokens []crypto.Token) {
	pool.mu.Lock()
	connections := make(map[*trusted.SignedConnection]struct{})
	for _, token := range tokens {
		if registered, ok := pool.indexation[token]; ok {
			for _, connection := range registered {
				connections[connection] = struct{}{}
			}
		}
	}
	pool.mu.Unlock()
	for conn := range connections {
		conn.Send(data)
	}
}
