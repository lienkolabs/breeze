package main

import "github.com/lienkolabs/breeze/crypto"

// SecrectVaultFile = path to vault with credentials for wallet dressing
type GatewayConfig struct {
	SecretVaultFile       string
	ValidatingNodeAddress string
	ValidatingNodeToken   *crypto.Token
}

func main() {

}
