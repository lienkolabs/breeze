package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/util"
)

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

func configureProvider(help util.Help) {
	config := GetConfig()
	if len(os.Args) < 4 {
		help.Command("config-provider")
		return
	}
	config.Provider = os.Args[2]
	token := crypto.TokenFromString(os.Args[3])
	if token.Equal(crypto.ZeroToken) {
		fmt.Println("cannot parse token")
		help.Command("config-provider")
		return
	}
	config.ProviderToken = os.Args[3]

	SaveConfig(config)
	fmt.Printf("\nConfiguration:\nBreeze gateway address: %v %v\nBreeze data provider address: %v %v\n\n", config.Gateway, config.GatewayToken, config.Provider, config.ProviderToken)
}

func configureGateway(help util.Help) {
	config := GetConfig()
	if len(os.Args) < 4 {
		help.Command("config-gateway")
		return
	}
	config.Gateway = os.Args[2]
	token := crypto.TokenFromString(os.Args[3])
	if token.Equal(crypto.ZeroToken) {
		fmt.Println("cannot parse token")
		help.Command("config-provider")
		return
	}
	config.GatewayToken = os.Args[3]
	SaveConfig(config)
	fmt.Printf("\nConfiguration:\nBreeze gateway address: %v %v\nBreeze data provider address: %v %v\n\n", config.Gateway, config.GatewayToken, config.Provider, config.ProviderToken)
}

func showConfig(help util.Help) {
	config := GetConfig()
	fmt.Printf("\nConfiguration:\nBreeze gateway address: %v %v\nBreeze data provider address: %v %v\n\n", config.Gateway, config.GatewayToken, config.Provider, config.ProviderToken)
}
