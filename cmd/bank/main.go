package main

import (
	"log"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network"
	"github.com/lienkolabs/breeze/network/echo"
)

type Config struct {
	Credentials         crypto.PrivateKey
	ServePort           int
	BlockServiceAddress string
	BlockServiveToken   *crypto.Token
}

func main() {
	var config Config
	configDB := echo.DBPoolConfig{
		Credentials: config.Credentials,
		Validator:   network.AcceptAllConnections,
		ServePort:   config.ServePort,
		Job:         make(chan *echo.NewIndexJob),
	}

	server, err := echo.NewDBPool(&configDB)
	if err != nil {
		log.Fatalf("could not instantiate index service: %v\n", err)
	}

	configBlock := echo.BlockListenerConfig{
		Credentials:         config.Credentials,
		BlockServiceAddress: config.BlockServiceAddress,
		BlockServiveToken:   *config.BlockServiveToken,
	}

	listener, err := echo.NewBlockListener(&configBlock)
	if err != nil {
		log.Fatalf("could not connect to block privider: %v\n", err)
	}
	if server != nil && listener != nil {
		return
	}
}
