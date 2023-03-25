package consensus

import (
	"time"

	"github.com/lienkolabs/breeze/core/block"
	"github.com/lienkolabs/breeze/core/crypto"
	"github.com/lienkolabs/breeze/core/state"
)

type BlockChain struct {
	GenesisTime     time.Time
	TotalStake      uint64
	Epoch           uint64
	CurrentState    *state.State
	RecentBlocks    SignedBlocks
	CandidateBlocks map[uint64]SignedBlocks
}

func (b *BlockChain) GetLastCheckpoint() *Checkpoint {
	starting := b.CurrentState.Epoch
	if len(b.RecentBlocks) == 0 || b.RecentBlocks[0].Block.Epoch() != starting+1 {
		return &Checkpoint{
			Validator: &block.MutatingState{
				State:     b.CurrentState,
				Mutations: state.NewMutation(),
			},
			CheckpointEpoch: b.CurrentState.Epoch,
		}
	}
	sequential := make([]*block.Block, 0)
	for _, block := range b.RecentBlocks {
		if block.Block.Epoch() != starting+1 {
			break
		}
		starting += 1
		sequential = append(sequential, block.Block)
	}
	return &Checkpoint{
		Validator: &block.MutatingState{
			State:     b.CurrentState,
			Mutations: block.GroupBlockMutations(sequential),
		},
		CheckpointEpoch: sequential[len(sequential)-1].Epoch(),
		CheckpointHash:  sequential[len(sequential)-1].Hash,
	}
}

func NewGenesisBlockChain(token crypto.Token) *BlockChain {
	state := state.NewGenesisStateWithToken(token)
	chain := BlockChain{
		GenesisTime:     time.Now(),
		TotalStake:      1000000,
		Epoch:           0,
		CurrentState:    state,
		RecentBlocks:    make(SignedBlocks, 0),
		CandidateBlocks: make(map[uint64]SignedBlocks),
	}
	return &chain
}
