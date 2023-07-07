package echo

import (
	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network/trusted"
)

type Listener struct {
	Connection *trusted.SignedConnection
	Incoming   chan []byte
}

func NewListener(credentials crypto.PrivateKey, address string, token crypto.Token) (*Listener, error) {
	conn, err := trusted.Dial(address, credentials, token)
	listener := &Listener{
		Connection: conn,
		Incoming:   make(chan []byte),
	}
	if err != nil {
		return nil, err
	}

	messages := make(chan []byte)
	shutdown := make(chan struct{})

	live := true

	go func() {
		for {
			msg, err := conn.Read()
			if err != nil {
				if live {
					live = false
					shutdown <- struct{}{}
				}
				return
			}
			messages <- msg
		}
	}()

	// close broken connections and process subscribe instructions
	go func() {
		for {
			select {
			case <-shutdown:
				if live {
					listener.Connection.Shutdown()
				}
				live = false
				return
			case msg := <-messages:
				listener.NewMessage(msg)
			}
		}
	}()
	return listener, nil
}

func (l *Listener) Shutdown() {

}

func (l *Listener) NewMessage(msg []byte) {
	switch msg[0] {
	case actionMsg:
	case nextBlockMsg:
	case sealBLockMsg:
	case commitBlockMsg:
	case rolloverBlockMsg:
	}
}
