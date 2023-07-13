package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network/echo"
	"github.com/lienkolabs/breeze/protocol/actions"
	"github.com/lienkolabs/breeze/util"
	"github.com/lienkolabs/breeze/vault"
)

type Wallet struct {
	vault  *vault.SecureVault
	data   *os.File
	config Config
}

func (w *Wallet) Sync(epoch uint64, tokens ...crypto.Token) {
	client := echo.DBClientConfig{
		ProviderAddress: w.config.Provider,
		ProviderToken:   crypto.TokenFromString(w.config.ProviderToken),
		Credentials:     w.vault.SecretKey,
		KeepAlive:       false,
	}
	provider, err := echo.NewDBClient(client)
	if err != nil {
		log.Fatalf("could not connect to provider: %v", err)
	}
	provider.Subscribe(epoch, false, tokens...)
	receiveAction := make(chan []byte)
	for {
		provider.Receive(receiveAction)
		action := <-receiveAction
		if len(action) == 0 {
			break
		}
		bytes := make([]byte, 0)
		util.PutUint16(uint16(len(bytes)), &bytes)
		bytes = append(bytes, action...)
		if n, err := w.data.Write(bytes); n != len(bytes) {
			log.Fatalf("could not write on data file: %v", err)
		}
	}
}

func (w *Wallet) GetEpochFromProvider() uint64 {
	client := echo.DBClientConfig{
		ProviderAddress: w.config.Provider,
		ProviderToken:   crypto.TokenFromString(w.config.ProviderToken),
		Credentials:     w.vault.SecretKey,
		KeepAlive:       false,
	}
	provider, err := echo.NewDBClient(client)
	if err != nil {
		log.Fatalf("could not connect to provider to get current epoch: %v", err)
	}
	provider.Shutdown()
	return provider.Epoch
}

func (w *Wallet) Send(action actions.Action) {
	token := crypto.TokenFromString(w.config.GatewayToken)
	if token.Equal(crypto.ZeroToken) {
		log.Fatalf("invalid gateway token\n")
	}
	server, err := echo.NewActionServer(w.config.Gateway, w.vault.SecretKey, token)
	if err != nil {
		log.Fatalf("could not connect to gatewat: %v\n", err)
	}
	server.Send(action.Serialize())
	server.Shutdown()
}

func (w *Wallet) GetBalances() map[crypto.Hash]uint64 {
	balances := make(map[crypto.Hash]uint64)
	position := int64(8)
	size := make([]byte, 1)
	for {
		if nbytes, _ := w.data.ReadAt(size, position); nbytes != 1 {
			return balances
		}
		position += 1
		data := make([]byte, int(size[0]))
		nbytes, err := w.data.ReadAt(size, position)
		if nbytes != len(data) || err == io.EOF {
			return balances
		}
		position += int64(nbytes)
		action := actions.ParseAction(data)
		payments := action.Payments()
		for _, w := range payments.Credit {
			balance := balances[w.Account]
			balances[w.Account] = balance + w.FungibleTokens
		}
		for _, w := range payments.Debit {
			balance := balances[w.Account]
			balances[w.Account] = balance - w.FungibleTokens
		}
	}
}

func (w *Wallet) ResetEpoch(epoch uint64) {
	bytes := make([]byte, 0)
	util.PutUint64(epoch, &bytes)
	if n, err := w.data.WriteAt(bytes, 0); n != 8 || err != nil {
		log.Fatalf("could not write on data file: %v", err)
	}
}

func (w *Wallet) Append(action []byte) {
	bytes := make([]byte, 0)
	util.PutUint16(uint16(len(action)), &bytes)
	bytes = append(bytes, action...)
	if _, err := w.data.Write(bytes); err != nil {
		log.Fatalf("could not write on data file: %v", err)
	}
}

func Help() {
	fmt.Print(helpdoc)
}

type Config struct {
	Gateway       string
	GatewayToken  string
	Provider      string
	ProviderToken string
}

func GetNewConfig() Config {
	fmt.Println("No condiguration found.")
	fmt.Println("Breeze Gateway:")
	var gateway string
	fmt.Scanln(&gateway)
	fmt.Println("Breeze Data Provider:")
	var provider string
	fmt.Scanln(&provider)
	return Config{
		Gateway:  gateway,
		Provider: provider,
	}
}

func SaveConfig(config Config) {
	path := util.DefaultHomeDir()
	path = filepath.Join(path, ".wallets")
	util.CreateFolderIfNotExists(path)
	path = filepath.Join(path, "config.json")
	bytes, err := json.Marshal(config)
	if err != nil {
		log.Fatalf("could not marshall config json: %v", err)
	}
	file, err := os.Create(path)
	if err != nil {
		log.Fatalf("could not create/truncate config file: %v", err)
	}
	if _, err := file.Write(bytes); err != nil {
		log.Fatalf("could not write on config file: %v", err)
	}
}

func GetConfig() Config {
	path := util.DefaultHomeDir()
	path = filepath.Join(path, ".wallets", "config.json")
	exists := util.FileExists(path)
	if !exists {
		config := GetNewConfig()
		SaveConfig(config)
		return config
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("could not read configuration file: %v", err)
	}
	var config Config
	json.Unmarshal(bytes, &config)
	return config
}

func OpenWallet() *Wallet {
	path := util.DefaultHomeDir()
	path = filepath.Join(path, ".wallets")
	util.CreateFolderIfNotExists(path)
	vault := util.OpenVault(filepath.Join(path, "secure.dat"))
	if vault == nil {
		log.Fatal("could not crete cault")
	}
	file, exists := util.CreateFileIfNotExists(".wallets", "actions.dat")
	if !exists {
		epoch := make([]byte, 8)
		if n, err := file.Write(epoch); n != 8 || err != nil {
			log.Fatalf("could not write on wallet data file: %v", err)
		}
	}
	return &Wallet{
		vault: vault,
		data:  file,
	}
}

func main() {
	if len(os.Args) == 1 {
		Help()
		return
	}
	config := GetConfig()
	if os.Args[1] == "show-config" {
		fmt.Printf("\nConfiguration:\nBreeze gateway address: %v\nBreeze data provider address: %v\n\n", config.Gateway, config.Provider)
		return
	}
	if os.Args[1] == "config-gateway" {
		if len(os.Args) < 3 {
			fmt.Print(helpconfiggateway)
			return
		}
		config.Gateway = os.Args[2]
		SaveConfig(config)
		fmt.Printf("\nConfiguration:\nBreeze gateway address: %v\nBreeze data provider address: %v\n\n", config.Gateway, config.Provider)
		return
	}
	if os.Args[1] == "config-provider" {
		if len(os.Args) < 3 {
			fmt.Print(helpconfigprovider)
			return
		}
		config.Provider = os.Args[2]
		SaveConfig(config)
		fmt.Printf("\nConfiguration:\nBreeze gateway address: %v\nBreeze data provider address: %v\n\n", config.Gateway, config.Provider)
		return
	}
	wallet := OpenWallet()
	if wallet == nil {
		log.Fatal("could not open/create wallet")
	}
	if os.Args[1] == "create-wallet" {
		token, _ := wallet.vault.GenerateNewKey()
		fmt.Printf("\nNew Token: %v\n\n", token)
		return
	}
	if os.Args[1] == "show-wallets" {
		fmt.Println("Token                                                               Balance")
		fmt.Println("--------------------------------------------------------------------------------")
		balances := wallet.GetBalances()
		for token, _ := range wallet.vault.Secrets {
			balance := balances[crypto.HashToken(token)]
			fmt.Printf("%v   %v\n", token, balance)
		}

		return
	}
	if os.Args[1] == "transfer" {
		if len(os.Args) < 5 {
			fmt.Print(helptransfer)
			return
		}
		from := crypto.TokenFromString(os.Args[2])
		if from.Equal(crypto.ZeroToken) {
			fmt.Println("invalid from-token")
			fmt.Print(helptransfer)
			return
		}
		credentails, ok := wallet.vault.Secrets[from]
		if !ok {
			fmt.Println("dont know secret for from-token")
			fmt.Print(helptransfer)
			return
		}
		qty, _ := strconv.Atoi(os.Args[3])
		if qty <= 0 {
			fmt.Println("invalid quantity")
			fmt.Print(helptransfer)
			return
		}
		to := crypto.TokenFromString(os.Args[4])
		if to.Equal(crypto.ZeroToken) {
			fmt.Println("invalid to-token")
			fmt.Print(helptransfer)
			return
		}
		transfer := actions.Transfer{
			TimeStamp: wallet.GetEpochFromProvider(),
			From:      from,
			To: []crypto.TokenValue{
				{Token: to, Value: uint64(qty)},
			},
		}
		if len(os.Args) >= 6 {
			fee, _ := strconv.Atoi(os.Args[5])
			if fee <= 0 {
				fmt.Println("invalid fee")
				fmt.Print(helptransfer)
				return
			}
			transfer.Fee = uint64(fee)
		}
		transfer.Sign(credentails)

	}

}
