package state

import (
	"github.com/lienkolabs/breeze/core/crypto"
)

type State struct {
	Epoch    uint64
	Wallets  *Wallet // Available tokens per hash of crypto key
	Deposits *Wallet // Available stakes per hash of crypto key
}

func NewGenesisState() (*State, crypto.PrivateKey) {
	pubKey, prvKey := crypto.RandomAsymetricKey()
	state := State{
		Epoch:    0,
		Wallets:  NewMemoryWalletStore(0, 8),
		Deposits: NewMemoryWalletStore(0, 8),
	}
	state.Wallets.Credit(pubKey, 1e6)
	state.Deposits.Credit(pubKey, 1e6)
	return &state, prvKey
}

func NewGenesisStateWithToken(token crypto.Token) *State {
	state := State{
		Epoch:    0,
		Wallets:  NewMemoryWalletStore(0, 8),
		Deposits: NewMemoryWalletStore(0, 8),
	}
	state.Wallets.Credit(token, 1e6)
	state.Deposits.Credit(token, 1e6)
	return &state
}
