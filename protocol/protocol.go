package protocol

import (
	"log"

	"github.com/lienkolabs/breeze/crypto"

	"github.com/lienkolabs/breeze/protocol/state"
)

func Genesis(credentials crypto.PrivateKey) *ValidatorNode {
	token := credentials.PublicKey()
	genesis := state.NewGenesisStateWithToken(token)
	chain := Blockchain{
		LastCommitEpoch: 0,
		LastCommitHash:  crypto.HashToken(token),
		CommitState:     genesis,
		Uncommitted:     make([]*Block, 0),
	}
	return &ValidatorNode{
		Credentials: credentials,
		Chain:       &chain,
	}
}

type ValidatorNode struct {
	Credentials crypto.PrivateKey
	Chain       *Blockchain
}

type Blockchain struct {
	LastCommitEpoch uint64
	LastCommitHash  crypto.Hash
	CommitState     *state.State
	Uncommitted     []*Block
}

func (v *ValidatorNode) IncorporateBlock(block *Block) {
	v.Chain.Uncommitted = append(v.Chain.Uncommitted, block)
}

func (v *ValidatorNode) Commit(epoch uint64, hash crypto.Hash) bool {
	block := v.Chain.Uncommitted[0]
	if epoch != v.Chain.LastCommitEpoch+1 || block.epoch != epoch || (!block.Hash.Equal(hash)) {
		return false
	}
	v.Chain.CommitState.IncorporateMutations(block.blockMutations)
	v.Chain.LastCommitEpoch = epoch
	v.Chain.LastCommitHash = hash
	v.Chain.Uncommitted = v.Chain.Uncommitted[1:]
	return true
}

func (v *ValidatorNode) NextBlock(against, epoch uint64) *Block {
	var validator *state.MutatingState
	var parentHash crypto.Hash
	if against == v.Chain.LastCommitEpoch {
		validator = &state.MutatingState{
			State: v.Chain.CommitState,
		}
		parentHash = v.Chain.LastCommitHash
	} else {
		parentCount := int(against) - int(v.Chain.LastCommitEpoch)
		if parentCount > len(v.Chain.Uncommitted) {
			log.Fatalf("unexpected error: checkpoint beyond known block")
		}
		mutations := make([]*state.Mutation, parentCount)
		for n := 0; n < parentCount; n++ {
			mutations[n] = v.Chain.Uncommitted[n].blockMutations
		}
		validator = &state.MutatingState{
			State:     v.Chain.CommitState,
			Mutations: state.GroupBlockMutations(mutations),
		}
		parentHash = v.Chain.Uncommitted[parentCount].Hash
	}
	block := NewBlock(parentHash, against, epoch, v.Credentials.PublicKey(), validator)
	return block
}

func (c Blockchain) Commit(epoch uint64) bool {
	if epoch != c.LastCommitEpoch+1 {
		return false
	}
	return true
}
