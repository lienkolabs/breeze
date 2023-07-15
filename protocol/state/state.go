package state

import (
	"fmt"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/protocol/chain"
)

type State struct {
	Epoch    uint64
	Wallets  *Wallet // Available tokens per hash of crypto key
	Deposits *Wallet // Available stakes per hash of crypto key
}

func (s *State) NewMutations() chain.Mutations {
	return NewMutations(s.Epoch + 1)
}

func (s *State) Validator(mutations chain.Mutations, epoch uint64) chain.MutatingState {
	m, ok := mutations.(*Mutations)
	if !ok {
		return nil
	}
	return &MutatingState{
		State:     s,
		mutations: m,
	}
}

func (s *State) Incorporate(v chain.MutatingState, publisher crypto.Token) {
	ms, ok := v.(*MutatingState)
	if !ok {
		return
	}
	publisherHash := crypto.HashToken(publisher)
	if delta, ok := ms.mutations.DeltaWallets[publisherHash]; ok {
		ms.mutations.DeltaWallets[publisherHash] = delta + int(ms.FeesCollected)
	} else {
		ms.mutations.DeltaWallets[publisherHash] = int(ms.FeesCollected)
	}
	s.IncorporateMutations(ms.mutations)
}

func (s *State) Shutdown() {
	s.Wallets.Close()
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

func NewGenesisStateWithToken(token crypto.Token, filePath string) *State {
	var state State
	if filePath == "" {
		state = State{
			Epoch:    0,
			Wallets:  NewMemoryWalletStore(0, 8),
			Deposits: NewMemoryWalletStore(0, 8),
		}
	} else {
		state = State{
			Epoch:    0,
			Wallets:  NewFileWalletStore(fmt.Sprintf("%vwallet.dat", filePath), 0, 8),
			Deposits: NewFileWalletStore(fmt.Sprintf("%vdeposit.dat", filePath), 0, 8),
		}

	}

	state.Wallets.Credit(token, 1e9)
	state.Deposits.Credit(token, 1e9)
	return &state
}

func (s *State) IncorporateMutations(m *Mutations) {
	for hash, delta := range m.DeltaWallets {
		if delta > 0 {
			s.Wallets.CreditHash(hash, uint64(delta))
		} else if delta < 0 {
			s.Wallets.DebitHash(hash, uint64(-delta))
		}
	}
	for hash, delta := range m.DeltaDeposits {
		if delta > 0 {
			s.Deposits.CreditHash(hash, uint64(delta))
		} else if delta < 0 {
			s.Deposits.DebitHash(hash, uint64(-delta))
		}
	}
}
