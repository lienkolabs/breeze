package block

import (
	"github.com/lienkolabs/breeze/core/crypto"
	"github.com/lienkolabs/breeze/core/state"
)

type MutatingState struct {
	State     *state.State
	Mutations *state.Mutation
}

// Balance returns the balance of fungible tokens associated to the hash.
// It returns zero if the hash is not found.
func (c *MutatingState) balance(hash crypto.Hash) uint64 {
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
