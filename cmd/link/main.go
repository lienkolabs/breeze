// Link provides a simple implementation of a service that offers the routing
// fee payment of breeze void instructions to the breeze network. Third parties
// might rely in these kind of services for smooth access into the breeze
// network.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network"
	"github.com/lienkolabs/breeze/network/echo"
	"github.com/lienkolabs/breeze/protocol/actions"
	"github.com/lienkolabs/breeze/util"
)

// SecrectVaultFile = path to vault with credentials for wallet dressing
type GatewayConfig struct {
	SecretVaultFile       string
	ValidatingNodeAddress string
	ValidatingNodeToken   crypto.Token
	Credentials           crypto.PrivateKey
	GatewayPort           int
}

type Config struct {
	GatewayPort       int
	BreezeNodeAddress string
	BreezeNodeToken   string
	SecureVaultPath   string
	NodeToken         string
}

type Gateway struct {
	listener *echo.Gateway
	shutdown chan chan struct{}
}

func (g *Gateway) Stats() []echo.Stats {
	return g.listener.Stats()
}

func (g *Gateway) Shutdown() {
	resp := make(chan struct{})
	g.shutdown <- resp
	<-resp
}

func NewGatway(config GatewayConfig, fee uint64) *Gateway {
	walletToken := config.Credentials.PublicKey()
	server, err := echo.NewActionServer(config.ValidatingNodeAddress, config.Credentials, config.ValidatingNodeToken)
	if err != nil {
		log.Fatalf("could not connect to breeze network: %v", err)
	}

	pipe := make(chan []byte)
	shutdown := make(chan chan struct{})

	listener, err := echo.NewActionsGateway(config.GatewayPort, config.Credentials, network.AcceptAllConnections, pipe)
	if err != nil {
		log.Fatalf("could not open port for listening connections: %v", err)
	}

	go func() {
		for {
			select {
			case done := <-shutdown:
				server.Shutdown()
				done <- struct{}{}
				return
			case msg := <-pipe:
				void := actions.ParseVoid(msg)
				if void != nil {
					void.Wallet = walletToken
					void.Fee = fee
					void.Sign(config.Credentials)
					server.Send(void.Serialize())
				}
			}
		}
	}()
	return &Gateway{
		listener: listener,
		shutdown: shutdown,
	}
}

func main() {
	var config Config
	if len(os.Args) < 2 {
		log.Fatalln("usage: breeze path-to-config-file.json")
	}
	util.ReadConfigFile(os.Args[1], &config)
	var gateway GatewayConfig
	gateway.ValidatingNodeAddress = config.BreezeNodeAddress
	gateway.ValidatingNodeToken = crypto.TokenFromString(config.BreezeNodeToken)
	if gateway.ValidatingNodeToken.Equal(crypto.ZeroToken) {
		log.Fatalln("invalid node token in config file")
	}
	credentials, isNew := util.GetOrSetCredentialsFromVault(config.SecureVaultPath)
	if isNew != "" {
		fmt.Println(isNew)
		return
	}
	token := crypto.TokenFromString(config.NodeToken)
	if !token.Equal(credentials.PublicKey()) {
		log.Fatalln("vault credentials does not match node token")
	}
	gateway.Credentials = credentials
	gateway.GatewayPort = config.GatewayPort

	running := NewGatway(gateway, 1)
	done := util.ShutdownEvents()
	<-done
	running.Shutdown()
}
