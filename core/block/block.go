package block

import (
	"fmt"
	"time"

	"github.com/lienkolabs/breeze/core/crypto"
	"github.com/lienkolabs/breeze/core/state"
	"github.com/lienkolabs/breeze/core/transactions"
	"github.com/lienkolabs/breeze/core/util"
)

type Block struct {
	epoch         uint64
	Parent        crypto.Hash
	CheckPoint    uint64
	Publisher     crypto.Token
	PublishedAt   time.Time
	Instructions  [][]byte
	Hash          crypto.Hash
	FeesCollected uint64
	Signature     crypto.Signature
	validator     *MutatingState
	mutations     *state.Mutation
}

func NewBlock(parent crypto.Hash, checkpoint, epoch uint64, publisher crypto.Token, validator *MutatingState) *Block {
	return &Block{
		Parent:       parent,
		epoch:        epoch,
		CheckPoint:   checkpoint,
		Publisher:    publisher,
		Instructions: make([][]byte, 0),
		validator:    validator,
		mutations:    state.NewMutation(),
	}
}

func (b *Block) Incorporate(instruction transactions.Transaction) bool {
	payments := instruction.Payments()
	if !b.CanPay(payments) {
		return false
	}
	b.TransferPayments(payments)
	b.Instructions = append(b.Instructions, instruction.Serialize())
	return true
}

func (b *Block) CanPay(payments *transactions.Payment) bool {
	for _, debit := range payments.Debit {
		existingBalance := b.validator.balance(debit.Account)
		delta := b.mutations.DeltaBalance(debit.Account)
		if int(existingBalance) < int(debit.FungibleTokens)+delta {
			return false
		}
	}
	return true
}

func (b *Block) CanWithdraw(hash crypto.Hash, value uint64) bool {
	existingBalance := b.validator.balance(hash)
	return value < existingBalance
}

func (b *Block) Deposit(hash crypto.Hash, value uint64) {
	if old, ok := b.validator.Mutations.DeltaDeposits[hash]; ok {
		b.validator.Mutations.DeltaDeposits[hash] = old + int(value)
		return
	}
	b.validator.Mutations.DeltaDeposits[hash] = int(value)
}

func (b *Block) TransferPayments(payments *transactions.Payment) {
	for _, debit := range payments.Debit {
		if delta, ok := b.mutations.DeltaWallets[debit.Account]; ok {
			b.mutations.DeltaWallets[debit.Account] = delta - int(debit.FungibleTokens)
		} else {
			b.mutations.DeltaWallets[debit.Account] = -int(debit.FungibleTokens)
			// fmt.Println(debit.Account, debit.FungibleTokens)
		}
	}
	for _, credit := range payments.Credit {
		if delta, ok := b.mutations.DeltaWallets[credit.Account]; ok {
			b.mutations.DeltaWallets[credit.Account] = delta + int(credit.FungibleTokens)
		} else {
			b.mutations.DeltaWallets[credit.Account] = int(credit.FungibleTokens)
		}
	}
}

func (b *Block) Balance(hash crypto.Hash) uint64 {
	return b.validator.balance(hash)
}

func (b *Block) AddFeeCollected(value uint64) {
	b.FeesCollected += value
}

func (b *Block) Epoch() uint64 {
	return b.epoch
}

func (b *Block) Sign(token crypto.PrivateKey) {
	b.Signature = token.Sign(b.serializeWithoutSignature())
}

func (b *Block) Serialize() []byte {
	bytes := b.serializeWithoutSignature()
	util.PutSignature(b.Signature, &bytes)
	return bytes
}

func (b *Block) serializeWithoutSignature() []byte {
	bytes := make([]byte, 0)
	util.PutUint64(b.epoch, &bytes)
	util.PutByteArray(b.Parent[:], &bytes)
	util.PutUint64(b.CheckPoint, &bytes)
	util.PutByteArray(b.Publisher[:], &bytes)
	util.PutTime(b.PublishedAt, &bytes)
	util.PutUint16(uint16(len(b.Instructions)), &bytes)
	for _, instruction := range b.Instructions {
		util.PutByteArray(instruction, &bytes)
	}
	util.PutByteArray(b.Hash[:], &bytes)
	util.PutUint64(b.FeesCollected, &bytes)
	return bytes
}

func ParseBlock(data []byte) *Block {
	position := 0
	block := Block{}
	block.epoch, position = util.ParseUint64(data, position)
	block.Parent, position = util.ParseHash(data, position)
	block.CheckPoint, position = util.ParseUint64(data, position)
	block.Publisher, position = util.ParseToken(data, position)
	block.PublishedAt, position = util.ParseTime(data, position)
	block.Instructions, position = util.ParseByteArrayArray(data, position)
	block.Hash, position = util.ParseHash(data, position)
	block.FeesCollected, position = util.ParseUint64(data, position)
	msg := data[0:position]
	block.Signature, _ = util.ParseSignature(data, position)
	if !block.Publisher.Verify(msg, block.Signature) {
		fmt.Println("wrong signature")
		return nil
	}
	block.mutations = state.NewMutation()
	return &block
}

func (b *Block) SetValidator(validator *MutatingState) {
	b.validator = validator
}

func GetBlockEpoch(data []byte) uint64 {
	if len(data) < 8 {
		return 0
	}
	epoch, _ := util.ParseUint64(data, 0)
	return epoch
}

func (b *Block) JSONSimple() string {
	bulk := &util.JSONBuilder{}
	bulk.PutUint64("epoch", b.epoch)
	bulk.PutHex("parent", b.Parent[:])
	bulk.PutUint64("checkpoint", b.CheckPoint)
	bulk.PutHex("publisher", b.Publisher[:])
	bulk.PutTime("publishedAt", b.PublishedAt)
	bulk.PutUint64("instructionsCount", uint64(len(b.Instructions)))
	bulk.PutHex("hash", b.Parent[:])
	bulk.PutUint64("feesCollectes", b.FeesCollected)
	bulk.PutBase64("signature", b.Signature[:])
	return bulk.ToString()
}
