package state

import (
	"github.com/lienkolabs/breeze/crypto"
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

func GroupBlockMutations(mutations []*Mutation) *Mutation {
	grouped := NewMutation()
	for _, mutation := range mutations {
		for acc, balance := range mutation.DeltaWallets {
			if oldBalance, ok := grouped.DeltaWallets[acc]; ok {
				grouped.DeltaWallets[acc] = oldBalance + balance
			} else {
				grouped.DeltaWallets[acc] = balance
			}
		}
		for acc, balance := range mutation.DeltaDeposits {
			if oldBalance, ok := grouped.DeltaDeposits[acc]; ok {
				grouped.DeltaDeposits[acc] = oldBalance + balance
			} else {
				grouped.DeltaDeposits[acc] = balance
			}
		}
	}
	return grouped
}

type MutatingState struct {
	State     *State
	Mutations *Mutation
}

func (c *MutatingState) Balance(hash crypto.Hash) uint64 {
	_, balance := c.State.Wallets.BalanceHash(hash)
	if c.Mutations == nil {
		return balance
	}
	delta := c.Mutations.DeltaBalance(hash)
	if delta < 0 {
		balance = balance - uint64(-delta)
	} else {
		balance = balance + uint64(delta)
	}
	return balance
}
