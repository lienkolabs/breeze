package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/lienkolabs/breeze/vault"
	"golang.org/x/term"
)

type Configuration struct {
	GatewayPort        int    `json:"gatewayPort"`
	BlockBroadcastPort int    `json:"blockBroadcastPort"`
	WalletDataPath     string `json:"walletDataPath"`
	SecureVaultPath    string `json:"secureVaultPath"`
	GenesisToken       string `json:"genesisToken"`
}

func main() {
	var config Configuration
	if len(os.Args) < 2 {
		log.Fatalln("usage: breeze path-to-config-file.json")
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalf("could not read config file: %v\n", err)
	}
	if err := json.Unmarshal(data, &config); err != nil {
		log.Fatalf("could not read config file: %v\n", err)
	}
	if config.GatewayPort == 0 || config.BlockBroadcastPort == 0 {
		log.Fatalf("invalid ports in configuration\n")
	}
	fmt.Print("Enter passphrase to secure validator credentials:")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatalf("could not read password: %v", err)
	}
	password := string(bytePassword)
	fmt.Println(password)
	vault := vault.NewSecureVault(password, config.SecureVaultPath)
	credential := vault.SecretKey
	fmt.Printf("validator token: %v\n", credential.PublicKey())

}
