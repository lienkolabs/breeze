package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

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
	w.config = GetConfig()
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
		if parser := actions.ParseAction(action); parser != nil {
			util.PrintJson(parser)
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
	util.PrintJson(action)
	server.Send(action.Serialize())
	server.Shutdown()
}

func (w *Wallet) GetActions() []actions.Action {
	all := make([]actions.Action, 0)
	position := int64(8)
	size := make([]byte, 1)
	for {
		if nbytes, _ := w.data.ReadAt(size, position); nbytes != 1 {
			return all
		}
		position += 1
		data := make([]byte, int(size[0]))
		nbytes, err := w.data.ReadAt(size, position)
		if nbytes != len(data) || err == io.EOF {
			return all
		}
		position += int64(nbytes)
		action := actions.ParseAction(data)
		all = append(all, action)
	}
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

func (w *Wallet) ReadEpoch() uint64 {
	bytes := make([]byte, 8)
	if n, err := w.data.ReadAt(bytes, 0); n != 8 || err != nil {
		log.Fatalf("could not write on data file: %v", err)
	}
	epoch, _ := util.ParseUint64(bytes, 0)
	return epoch
}

func (w *Wallet) Append(action []byte) {
	bytes := make([]byte, 0)
	util.PutUint16(uint16(len(action)), &bytes)
	bytes = append(bytes, action...)
	if _, err := w.data.Write(bytes); err != nil {
		log.Fatalf("could not write on data file: %v", err)
	}
}

func createVault(h util.Help) {
	filePath, ok := flags["vault"]
	if !ok {
		fmt.Println("specify vault filename to be created.")
		h.Doc()
	}
	if util.FileExists(filePath) {
		fmt.Println("file already exists")
		os.Exit(0)
	}
	vault := util.OpenVault(filePath)
	fmt.Printf("vault %v create with the new token %v\n", filePath, vault.SecretKey.PublicKey())
	os.Exit(0)
}

func showToken(h util.Help) {
	var path string
	if filePath, ok := flags["vault"]; ok {
		path = filePath
	} else {
		path = util.DefaultHomeDir()
		path = filepath.Join(path, ".wallets", "secure.dat")
	}
	if !util.FileExists(path) {
		fmt.Println("vault file not found")
		os.Exit(0)
	}
	vault := util.OpenVault(path)
	if vault == nil {
		fmt.Println("could not open vault")
		os.Exit(0)
	}
	fmt.Printf("vault %v token:\n%v\n", path, vault.SecretKey.PublicKey())
	os.Exit(0)
}

func OpenWallet() *Wallet {
	var vault *vault.SecureVault
	if filePath, ok := flags["vault"]; ok {
		if !util.FileExists(filePath) {
			fmt.Printf("file not found: %v\n", filePath)
			os.Exit(0)
		}
		vault = util.OpenVault(filePath)
	} else {
		path := util.DefaultHomeDir()
		path = filepath.Join(path, ".wallets")
		util.CreateFolderIfNotExists(path)
		vault = util.OpenVault(filepath.Join(path, "secure.dat"))

	}
	if vault == nil {
		log.Fatal("could not create vault")
	}
	file, exists := util.CreateFileIfNotExists(".wallets", "actions.dat")
	if !exists {
		epoch := make([]byte, 8)
		if n, err := file.Write(epoch); n != 8 || err != nil {
			log.Fatalf("could not write on wallet data file: %v", err)
		}
	}
	return &Wallet{
		vault:  vault,
		data:   file,
		config: GetConfig(),
	}
}

func createWallet(help util.Help) {
	wallet := OpenWallet()
	token, _ := wallet.vault.GenerateNewKey()
	fmt.Printf("\nNew Token: %v\n\n", token)
}

func showWallets(help util.Help) {
	wallet := OpenWallet()
	fmt.Println("Token                                                               Balance")
	fmt.Println("--------------------------------------------------------------------------------")
	balances := wallet.GetBalances()
	for token, _ := range wallet.vault.Secrets {
		balance := balances[crypto.HashToken(token)]
		fmt.Printf("%v   %v\n", token, balance)
	}
}

func sync(help util.Help) {
	wallet := OpenWallet()
	epoch := wallet.ReadEpoch()
	tokens := make([]crypto.Token, 0)
	for token, _ := range wallet.vault.Secrets {
		tokens = append(tokens, token)
	}
	wallet.Sync(epoch, tokens...)
}
