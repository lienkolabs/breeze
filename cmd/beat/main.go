package main

import (
	"fmt"
	"log"
	"os"

	"github.com/lienkolabs/breeze/consensus/poa"
	"github.com/lienkolabs/breeze/util"
)

type Configuration struct {
	GatewayPort        int    `json:"gatewayPort"`
	BlockBroadcastPort int    `json:"blockBroadcastPort"`
	WalletDataPath     string `json:"walletDataPath"`
	SecureVaultPath    string `json:"secureVaultPath"`
	NodeToken          string `json:"nodeToken"`
	//GenesisToken       string `json:"genesisToken"`
}

func main() {
	var config Configuration
	if len(os.Args) < 2 {
		log.Fatalln("usage: breeze path-to-config-file.json")
	}
	util.ReadConfigFile(os.Args[1], &config)
	if config.GatewayPort == 0 || config.BlockBroadcastPort == 0 {
		log.Fatalf("invalid ports in configuration\n")
	}
	if config.SecureVaultPath == "" {
		log.Fatalf("no path to secure vault specified in the configuration file\n")
	}
	credentials, isNew := util.GetOrSetCredentialsFromVault(config.SecureVaultPath)
	if isNew != "" {
		fmt.Println(isNew)
		return
	}
	if err := poa.NewProofOfAuthorityValidator(credentials, config.GatewayPort, config.BlockBroadcastPort, config.WalletDataPath); err != nil {
		log.Fatalf("could not initiate node: %v\n", err)
	}
	fmt.Printf("\nnode started\ntoken:%v", credentials.PublicKey())

	done := util.ShutdownEvents()

	if len(os.Args) >= 3 && os.Args[2] == "test" {
		go Simulation(credentials, config.GatewayPort)
	}
	<-done
}
