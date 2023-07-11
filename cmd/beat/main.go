package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lienkolabs/breeze/consensus/poa"
	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network/echo"
	"github.com/lienkolabs/breeze/protocol/actions"
	"github.com/lienkolabs/breeze/vault"
	"golang.org/x/term"
)

type Configuration struct {
	GatewayPort        int    `json:"gatewayPort"`
	BlockBroadcastPort int    `json:"blockBroadcastPort"`
	WalletDataPath     string `json:"walletDataPath"`
	SecureVaultPath    string `json:"secureVaultPath"`
	//GenesisToken       string `json:"genesisToken"`
}

func vaultExists(filepath string) bool {
	info, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
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
	if config.SecureVaultPath == "" {
		log.Fatalf("no path to secure vault specified in the configuration file\n")
	}
	var credentials crypto.PrivateKey
	if vaultExists(config.SecureVaultPath) {
		fmt.Print("Enter passphrase of secure vault:")
		bytePassword, _ := term.ReadPassword(int(syscall.Stdin))
		secrets := vault.OpenVaultFromPassword(bytePassword, config.SecureVaultPath)
		credentials = secrets.SecretKey
		secrets.Close()
	} else {
		fmt.Print("Enter passphrase for a new secure vault:")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatalf("could not read password: %v", err)
		}
		password := string(bytePassword)
		secrets := vault.NewSecureVault(password, config.SecureVaultPath)
		credentials = secrets.SecretKey
	}
	if err := poa.NewProofOfAuthorityValidator(credentials, config.GatewayPort, config.BlockBroadcastPort, config.WalletDataPath); err != nil {
		log.Fatalf("could not initiate node: %v\n", err)
	}
	fmt.Println("node started")
	c := make(chan os.Signal, 1) // we need to reserve to buffer size 1, so the notifier are not blocked
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	testShutdown := make(chan struct{})
	count := uint64(0)
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		for n := 0; n < 10; n++ {
			<-ticker.C
		}
		_, secret := crypto.RandomAsymetricKey()
		server, err := echo.NewActionServer(fmt.Sprintf("localhost:%v", config.GatewayPort), secret, credentials.PublicKey())
		if err != nil {
			log.Fatalf("could not connect to gateway: %v", err)
		}
		for {
			select {
			case <-ticker.C:
				count++
				address, _ := crypto.RandomAsymetricKey()
				transfer := actions.Transfer{
					TimeStamp: count / 10,
					From:      credentials.PublicKey(),
					To: []crypto.TokenValue{
						{
							Token: address,
							Value: 1,
						},
					},
					Reason: "Testing",
					Fee:    0,
				}
				transfer.Sign(credentials)
				server.Send(transfer.Serialize())
			case <-testShutdown:
				ticker.Stop()
				return

			}
		}
	}()

	for {
		<-c
		fmt.Println("shuting down")
		return
	}
}
