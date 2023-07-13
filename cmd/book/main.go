package main

import (
	"fmt"
	"log"
	"os"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network"
	"github.com/lienkolabs/breeze/network/echo"
	"github.com/lienkolabs/breeze/protocol/actions"
	"github.com/lienkolabs/breeze/store"
	"github.com/lienkolabs/breeze/util"
)

type Config struct {
	ServePort           int
	BlockServiceAddress string
	BlockServiveToken   string
	FileNameTemplate    string
	NodeToken           string
	SecureVaultPath     string
}

func main() {
	var config Config
	if len(os.Args) < 2 {
		log.Fatalln("usage: breeze path-to-config-file.json")
	}
	util.ReadConfigFile(os.Args[1], &config)

	if config.SecureVaultPath == "" {
		log.Fatalf("no path to secure vault specified in the configuration file\n")
	}
	credentials, isNew := util.GetOrSetCredentialsFromVault(config.SecureVaultPath)
	if isNew != "" {
		fmt.Println(isNew)
		return
	}
	token := crypto.TokenFromString(config.NodeToken)
	if !token.Equal(credentials.PublicKey()) {
		log.Fatalf("credentials on security vault does not match config node token\n")
	}

	jobs := make(chan *echo.NewIndexJob)
	configDB := echo.DBPoolConfig{
		Credentials: credentials,
		Validator:   network.AcceptAllConnections,
		ServePort:   config.ServePort,
		Job:         jobs,
	}

	server, err := echo.NewDBPool(&configDB)
	if err != nil {
		log.Fatalf("could not instantiate index service: %v\n", err)
	}

	configBlock := echo.BlockListenerConfig{
		Credentials:         credentials,
		BlockServiceAddress: config.BlockServiceAddress,
		BlockServiveToken:   crypto.TokenFromString(config.BlockServiveToken),
	}

	listener, err := echo.NewBlockListener(&configBlock)
	if err != nil {
		log.Fatalf("could not connect to block privider: %v\n", err)
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
				fmt.Println(block.Epoch, len(block.Actions))
			case job := <-jobs:
				db.AppendJob(job)
			case confirm := <-shutdown:
				server.Shutdown()
				listener.Shutdown()
				db.Close()
				confirm <- struct{}{}
				return
			}
		}
	}()

	done := util.ShutdownEvents()
	<-done

}
