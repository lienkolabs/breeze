package echo

import (
	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network/trusted"
)

type ActionServer struct {
	conn *trusted.SignedConnection
}

func NewActionServer(gatewayAddress string, credentials crypto.PrivateKey, gatewayToken crypto.Token) (*ActionServer, error) {
	conn, err := trusted.Dial(gatewayAddress, credentials, gatewayToken)
	if err != nil {
		return nil, err
	}
	return &ActionServer{conn}, nil
}

func (a *ActionServer) Send(action []byte) {
	a.conn.Send(action)
}

func (a *ActionServer) Shutdown() {
	a.conn.Shutdown()
}
