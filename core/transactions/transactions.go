package transactions

import (
	"github.com/lienkolabs/breeze/core/crypto"
	"github.com/lienkolabs/breeze/core/util"
)

const (
	ITransfer byte = iota
	IDeposit
	IWithdraw
	IPublish
	iUnkown
)

type HashTransaction struct {
	Transaction Transaction
	Hash        crypto.Hash
}

type Wallet struct {
	Account        crypto.Hash
	FungibleTokens uint64
}

type Payment struct {
	Debit  []Wallet
	Credit []Wallet
}

type Transaction interface {
	Payments() *Payment
	Serialize() []byte
	Epoch() uint64
	Kind() byte
}

func NewPayment(debitAcc crypto.Hash, value uint64) *Payment {
	return &Payment{
		Debit:  []Wallet{{debitAcc, value}},
		Credit: []Wallet{},
	}
}

func (p *Payment) NewCredit(account crypto.Hash, value uint64) {
	for _, credit := range p.Credit {
		if credit.Account.Equal(account) {
			credit.FungibleTokens += value
			return
		}
	}
	p.Credit = append(p.Credit, Wallet{Account: account, FungibleTokens: value})
}

func (p *Payment) NewDebit(account crypto.Hash, value uint64) {
	for _, debit := range p.Debit {
		if debit.Account.Equal(account) {
			debit.FungibleTokens += value
			return
		}
	}
	p.Debit = append(p.Debit, Wallet{Account: account, FungibleTokens: value})
}

func ParseTransaction(data []byte) Transaction {
	if data[0] != 0 {
		return nil
	}
	switch data[1] {
	case ITransfer:
		return ParseTransfer(data)
	case IDeposit:
		return ParseDeposit(data)
	case IWithdraw:
		return ParseWithdraw(data)
	case IPublish:
		return ParsePublish(data)
	}
	return nil
}

func GetEpochFromByteArray(inst []byte) uint64 {
	epoch, _ := util.ParseUint64(inst, 2)
	return epoch
}