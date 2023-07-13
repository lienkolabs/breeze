package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/protocol/actions"
	"github.com/lienkolabs/breeze/util"
)

func deposit(help util.Help) {
	wallet := OpenWallet()
	if len(os.Args) < 4 {
		help.Command("deposit")
		return
	}
	from := crypto.TokenFromString(os.Args[2])
	if from.Equal(crypto.ZeroToken) {
		fmt.Println("invalid from-token")
		help.Command("deposit")
		return
	}
	credentails, ok := wallet.vault.Secrets[from]
	if !ok {
		fmt.Println("dont know secret for from-token")
		help.Command("deposit")
		return
	}
	qty, _ := strconv.Atoi(os.Args[3])
	if qty <= 0 {
		fmt.Println("invalid quantity")
		help.Command("deposit")
		return
	}
	deposit := actions.Deposit{
		TimeStamp: wallet.GetEpochFromProvider(),
		Token:     from,
		Value:     uint64(qty),
	}
	deposit.Sign(credentails)
	wallet.Send(&deposit)
}

func withdraw(help util.Help) {
	wallet := OpenWallet()
	if len(os.Args) < 4 {
		help.Command("withdraw")
		return
	}
	from := crypto.TokenFromString(os.Args[2])
	if from.Equal(crypto.ZeroToken) {
		fmt.Println("invalid deposit-token")
		help.Command("withdraw")
		return
	}
	credentails, ok := wallet.vault.Secrets[from]
	if !ok {
		fmt.Println("dont know secret for deposit-token")
		help.Command("withdraw")
		return
	}
	qty, _ := strconv.Atoi(os.Args[3])
	if qty <= 0 {
		fmt.Println("invalid quantity")
		help.Command("withdraw")
		return
	}
	withdraw := actions.Withdraw{
		TimeStamp: wallet.GetEpochFromProvider(),
		Token:     from,
		Value:     uint64(qty),
	}
	withdraw.Sign(credentails)
	wallet.Send(&withdraw)
}

func transfer(help util.Help) {
	wallet := OpenWallet()
	if len(os.Args) < 5 {
		help.Command("transfer")
		return
	}
	from := crypto.TokenFromString(os.Args[2])
	if from.Equal(crypto.ZeroToken) {
		fmt.Println("invalid from-token")
		help.Command("transfer")
		return
	}
	credentails, ok := wallet.vault.Secrets[from]
	if !ok {
		fmt.Println("dont know secret for from-token")
		help.Command("transfer")
		return
	}
	qty, _ := strconv.Atoi(os.Args[3])
	if qty <= 0 {
		fmt.Println("invalid quantity")
		help.Command("transfer")
		return
	}
	to := crypto.TokenFromString(os.Args[4])
	if to.Equal(crypto.ZeroToken) {
		fmt.Println("invalid to-token")
		help.Command("transfer")
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
			help.Command("transfer")
			return
		}
		transfer.Fee = uint64(fee)
	}
	transfer.Sign(credentails)
	wallet.Send(&transfer)
}
