package poa

import (
	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network/echo"
	"github.com/lienkolabs/breeze/network/trusted"
)

const ActionsGatewayPort = 3100
const BroadcastPoolPort = 3101

type Node struct {
	gateway       *echo.Gateway
	broadcast     *echo.BroadcastPool
	actionChannel chan []byte
}

func NewProofOfAuthorityNetwork(credentials crypto.PrivateKey) (*Node, error) {
	actions := make(chan []byte)
	gateway, err := echo.NewActionsGateway(ActionsGatewayPort, credentials, trusted.AcceptAllConnections, actions)
	if err != nil {
		return nil, err
	}
	pool, err := echo.NewBroadcastPool(credentials, trusted.AcceptAllConnections, BroadcastPoolPort)
	if err != nil {
		return nil, err
	}
	return &Node{
		gateway:       gateway,
		broadcast:     pool,
		actionChannel: actions,
	}, nil

}
