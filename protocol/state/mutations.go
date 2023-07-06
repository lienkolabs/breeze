package state

import (
	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/protocol/chain"
)

type Mutations struct {
	DeltaWallets  map[crypto.Hash]int
	DeltaDeposits map[crypto.Hash]int
}

func NewMutations() *Mutations {
	return &Mutations{
		DeltaWallets:  make(map[crypto.Hash]int),
		DeltaDeposits: make(map[crypto.Hash]int),
	}
}

func (m *Mutations) DeltaBalance(hash crypto.Hash) int {
	value := m.DeltaWallets[hash]
	return value
}

func (m *Mutations) Append(array []chain.Mutations) chain.Mutations {
	grouped := NewMutations()
	all := []*Mutations{m}
	if len(array) > 0 {
		for _, a := range array {
			if mutation, ok := a.(*Mutations); ok {
				all = append(all, mutation)
			}
		}
	}
	for _, mutations := range all {
		for hash, delta := range mutations.DeltaWallets {
			if value, ok := grouped.DeltaWallets[hash]; ok {
				grouped.DeltaWallets[hash] = value + delta
			} else {
				grouped.DeltaWallets[hash] = delta
			}
		}
		for hash, delta := range mutations.DeltaDeposits {
			if value, ok := grouped.DeltaDeposits[hash]; ok {
				grouped.DeltaDeposits[hash] = value + delta
			} else {
				grouped.DeltaDeposits[hash] = delta
			}
		}
	}
	return grouped
}
