package main

const helpdoc = `
safe is a tool for managing crypto keys for wallets in the breeze ecosystem. 

Usage: 

	safe <command> [arguments]

The commands are:

	show-wallets    show balances of wallet registered on the secure vault 
	sync            syncrhonizes information with the breeze network
	create-wallet   create new wallet key pair and show token
	config-gateway  define new gateway to breeze network
	config-provider define new information provider from the breeze network
	show-config     show configurations
	transfer        send simple transfer order to breeze network

Use "safe help <command>" for more information about a command. 
`

const helpconfiggateway = `
Usage: 

	safe config-gateway gateway-address

`

const helpconfigprovider = `
Usage: 

	safe config-provider gateway-address

`

const helptransfer = `
Usage: 

	safe transfer from-token quantity to-token [reason]

`
