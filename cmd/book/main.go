package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network"
	"github.com/lienkolabs/breeze/network/echo"
	"github.com/lienkolabs/breeze/protocol/actions"
	"github.com/lienkolabs/breeze/store"
)

type Config struct {
	Credentials         crypto.PrivateKey
	ServePort           int
	BlockServiceAddress string
	BlockServiveToken   *crypto.Token
	FileNameTemplate    string
}

func main() {
	jobs := make(chan *echo.NewIndexJob)
	var config Config
	configDB := echo.DBPoolConfig{
		Credentials: config.Credentials,
		Validator:   network.AcceptAllConnections,
		ServePort:   config.ServePort,
		Job:         jobs,
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

	db, err := store.NewDB(config.FileNameTemplate, actions.GetTokens)
	if err != nil {
		listener.Shutdown()
		server.Shutdown()
		log.Fatalf("could not launch block db instance: %v", err)
	}

	shutdown := make(chan chan struct{})

	go func() {
		for {
			select {
			case block := <-listener.Block:
				db.AppendBlock(block)
			case <-jobs:

			case confirm := <-shutdown:
				server.Shutdown()
				listener.Shutdown()
				db.Close()
				confirm <- struct{}{}
				return
			}
		}
	}()

	c := make(chan os.Signal, 1) // we need to reserve to buffer size 1, so the notifier are not blocked
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	confirm := make(chan struct{})
	shutdown <- confirm
	<-confirm
}
