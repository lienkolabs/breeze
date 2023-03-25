package state

import (
	"github.com/lienkolabs/breeze/core/crypto"
)

type Mutation struct {
	DeltaWallets  map[crypto.Hash]int
	DeltaDeposits map[crypto.Hash]int
}

func NewMutation() *Mutation {
	return &Mutation{
		DeltaWallets: make(map[crypto.Hash]int),
	}
}

func (m *Mutation) DeltaBalance(hash crypto.Hash) int {
	balance := m.DeltaWallets[hash]
	return balance
}

type MutatingState struct {
	State         *State
	DeltaWallets  map[crypto.Hash]int
	DeltaDeposits map[crypto.Hash]int
}
