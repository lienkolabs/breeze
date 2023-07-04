package echo

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network"
	"github.com/lienkolabs/breeze/network/trusted"
	"github.com/lienkolabs/breeze/util"
)

type NullSocialProtocol struct{}

func (n NullSocialProtocol) Validate(data []byte) bool {
	return true
}

func (n NullSocialProtocol) CommitBlock(epoch uint64, hash crypto.Hash) {
}

func (n NullSocialProtocol) NextBlock(epoch, checkpoint uint64, hash crypto.Hash, token crypto.Token, timestamp time.Time) {
}

type SocialProtocol interface {
	Validate([]byte) bool
	CommitBlock(uint64)
	NextBlock(uint64)
	RolloverBlock(uint64)
	Shutdown() error
}

func StartNode(social SocialProtocol, credentials crypto.PrivateKey, validator network.ValidateConnection, listenPort int) (*Node, error) {
	node := &Node{
		mu:       sync.Mutex{},
		outbound: make(map[crypto.Token]*listener),
		social:   social,
	}

	listeners, err := net.Listen("tcp", fmt.Sprintf(":%v", listenPort))
	if err != nil {
		return nil, err
	}

	listenersChan := make(chan trusted.Message)
	shutdownConnection := make(chan crypto.Token)

	// listen connecttions
	go func() {
		for {
			if conn, err := listeners.Accept(); err == nil {
				trustedConn, err := trusted.PromoteConnection(conn, credentials, validator)
				if err != nil {
					conn.Close()
				} else {
					node.mu.Lock()
					node.outbound[trustedConn.Token] = &listener{
						connection: trustedConn,
					}
					node.mu.Unlock()
					trustedConn.Listen(listenersChan, shutdownConnection)
				}
			} else {
				return
			}
		}
	}()

	// listen
	go func() {
		for {
			select {
			case token := <-shutdownConnection:
				node.mu.Lock()
				if conn, ok := node.outbound[token]; ok {
					conn.connection.Shutdown()
				}
				delete(node.outbound, token)
				node.mu.Unlock()
			case msg := <-listenersChan:
				if msg.Data[0] == subscribeMsg {
					if trusted, ok := node.outbound[msg.Token]; ok && len(msg.Data) == 5 {
						for n := 0; n < 4; n++ {
							trusted.code[n] = msg.Data[n+1]
						}
					}
				}
			}
		}
	}()

	return nil, nil
}

type listener struct {
	connection *trusted.SignedConnection
	code       ProtocolCode
}

type Node struct {
	mu       sync.Mutex
	social   SocialProtocol
	outbound map[crypto.Token]*listener
}

func (n *Node) ConntectTo(address string, token crypto.Token, credentials crypto.PrivateKey) error {
	receiving, err := trusted.Dial(address, credentials, token)
	if err != nil {
		return err
	}

	go func() {
		for {
			if msg, err := receiving.Read(); err == nil {
				n.Incorporate(msg)
			}
		}
	}()
	return nil
}

func (n *Node) CommitBlock(epoch uint64) {
	n.social.CommitBlock(epoch)
	msg := []byte{commitBlockMsg}
	util.PutUint64(epoch, &msg)
	n.Broadcast(msg)
}

func (n *Node) RolloverBlock(epoch uint64) {
	n.social.RolloverBlock(epoch)
	msg := []byte{rolloverBlockMsg}
	util.PutUint64(epoch, &msg)
	n.Broadcast(msg)
}

func (n *Node) NextBlock(epoch uint64) {
	n.social.NextBlock(epoch)
	msg := []byte{nextBlockMsg}
	util.PutUint64(epoch, &msg)
	n.Broadcast(msg)
}

func (n *Node) Broadcast(msg []byte) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, conn := range n.outbound {
		conn.connection.Send(msg)
	}
}

func (n *Node) Incorporate(msg []byte) {
	if len(msg) == 0 {
		return
	}
	if msg[0] == commitBlockMsg || msg[0] == rolloverBlockMsg || msg[0] == nextBlockMsg {
		if len(msg) != 9 {
			return
		}
		epoch, _ := util.ParseUint64(msg, 1)
		switch msg[0] {
		case nextBlockMsg:
			n.social.CommitBlock(epoch)
		case commitBlockMsg:
			n.social.CommitBlock(epoch)
		case rolloverBlockMsg:
			n.social.RolloverBlock(epoch)
		}
		n.Broadcast(msg)
		return
	}
	if msg[0] != socialMsg {
		return
	}
	if !n.social.Validate(msg[1:]) {
		return
	}
	n.Broadcast(msg)
}
