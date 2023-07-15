package main

import "github.com/lienkolabs/breeze/util"

var help = util.Help{
	Executable: "safe",
	Short:      "is a tool for managing crypto keys for wallets in the breeze ecosystem.",
	Flags:      map[string]string{"vault": "path"},
	Commands: map[string]util.CommandHelp{
		"show-wallets": {
			Usage:       "safe show-wallets",
			Short:       "show balances of wallet registered on the secure vault",
			Description: "",
			Execute:     showWallets,
		},

		"sync": {
			Usage:       "safe sync",
			Short:       "syncrhonizes information with the breeze network",
			Description: "",
			Execute:     sync,
		},

		"create-wallet": {
			Usage:       "safe create-wallets",
			Short:       "create new wallet key pair and show token",
			Description: "",
			Execute:     createWallet,
		},

		"config-gateway": {
			Usage:       "safe config-gateway gateway-address gateway-token",
			Short:       "define new gateway to breeze network",
			Description: "",
			Execute:     configureGateway,
		},

		"config-provider": {
			Usage:       "safe config-provider provider-address provider-token",
			Short:       "safe config-provider provider-address",
			Description: "",
			Execute:     configureProvider,
		},

		"show-config": {
			Usage:       "safe show-config",
			Short:       "show configurations",
			Description: "",
			Execute:     showConfig,
		},

		"transfer": {
			Usage:       "safe transfer from-token quantity to-token [reason]",
			Short:       "send simple transfer order to breeze network",
			Description: "",
			Execute:     transfer,
		},

		"deposit": {
			Usage:       "safe deposit from-token quantity",
			Short:       "deposit tokens for PoS collateral",
			Description: helpDeposit,
			Execute:     deposit,
		},

		"withdraw": {
			Usage:       "safe withdraw deposit-token quantity",
			Short:       "withdraw deposited tokens",
			Description: "",
			Execute:     withdraw,
		},

		"create-vault": {
			Usage:       "safe [--vault=<path>] create-vault filename",
			Short:       "create a new vault file with a private key",
			Description: "",
			Execute:     createVault,
		},

		"show-token": {
			Usage:       "safe [--vault=<path>] show-token filename",
			Short:       "create a new vault file with a private key",
			Description: "",
			Execute:     showToken,
		},

		"actions": {
			Usage:       "safe actions",
			Short:       "show all actions associated to tokens",
			Description: "",
			Execute:     history,
		},
	},
}

const helpDeposit = `Deposited tokens will remain frozen and cannot be used for transfers until 
withdrawn.

In order to candidate for participation within validator network the holder of
secrets associated with the token must run a validating node with annoucing the
deposited token and must provide a valid checksum for the state of the network
according to breeze protocol.`
