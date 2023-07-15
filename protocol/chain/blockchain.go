/*
	chain provides blockchain interface

Rules for the chain mechanism:

 1. blocks are proposed for a certain epoch and against a certain checkpoint
    prior to that epoch.
 2. the block associated to a checkpoint must be sealed, otherwise it is not a
    valid checkpoint. sealed blocks cannot append new actions. They are not
    considerer final because certain actions can be removed by the commit phase.
 2. actions for the block are temporarily validated against the state derived
    at the checkpoint epoch.
 3. blocks are sealed, a hash is calculated, and the hash is signed by the
    publisher of the block. the commit phase is done by every node
 4. blocks are commited with all transactions validated with the checkpoint of
    the epoch immediately before the block epoch. Actions that were approved as
    validated by the original checkpoint are marked as invalidated by the commit
    instruction.
*/
package chain

import (
	"errors"
	"time"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/protocol/actions"
)

const KeepLastN = 100

type MutatingState interface {
	Validate(msg []byte) bool
	Mutations() Mutations
	GetEpoch() uint64
}

type Mutations interface {
	Append([]Mutations) Mutations
	GetEpoch() uint64
}

type State interface {
	NewMutations() Mutations
	Validator(Mutations, uint64) MutatingState
	Incorporate(MutatingState, crypto.Token)
	Shutdown()
}

// Chain is a non-disputed block interface... one block proposed for each
// epoch, every block is sealed before the proposal of a new block.
// Final commit of blocks can be delayed and the chain might be asked to
// rollover to any epoch after the last commit epoch. disaster recovery,
// that means, the rollover before last commit epoch is not anticipated on the
// structure and must be implemented separatedly.
type Chain struct {
	Incorporated    *IncorporatedActions
	Credentials     crypto.PrivateKey
	LastCommitEpoch uint64
	LastCommitHash  crypto.Hash
	CommitState     State
	SealedBlocks    map[uint64]*Block
	LiveBlock       *Block
}

func (c *Chain) NewBlock(epoch, checkpoint uint64, publisher crypto.Token) (*Block, error) {
	if epoch <= c.LastCommitEpoch {
		return nil, errors.New("cannot replace commited block outside recovery mode")
	}
	parent, ok := c.SealedBlocks[checkpoint]
	if !ok {
		return nil, errors.New("cannot find referred checkpoint")
	}
	mutations := c.CommitState.NewMutations()
	if parent.Epoch >= c.LastCommitEpoch {
		mutations = mutations.Append([]Mutations{parent.Validator.Mutations()})
	}
	return &Block{
		Epoch:          epoch,
		CheckPoint:     checkpoint,
		CheckpointHash: parent.Hash,
		Proposer:       publisher,
		Actions:        make([][]byte, 0),
		Validator:      c.CommitState.Validator(mutations, checkpoint),
	}, nil
}

func (c *Chain) CommitNextBlock() bool {
	block, ok := c.SealedBlocks[c.LastCommitEpoch+1]
	if !ok {
		return false
	}
	if block.CheckPoint != c.LastCommitEpoch {
		validator := c.CommitState.Validator(nil, c.LastCommitEpoch)
		block.Revalidate(validator)
	}
	c.CommitState.Incorporate(block.Validator, block.Proposer)
	c.LastCommitEpoch = block.Epoch
	c.LastCommitHash = block.Hash
	return true
}

func (c *Chain) Validate(action []byte) bool {
	epoch := actions.GetEpochFromByteArray(action)
	if epoch == 0 || epoch > c.LiveBlock.Epoch || (epoch+MaxProtocolEpoch < c.LiveBlock.Epoch) {
		return false
	}
	hash := crypto.Hasher(action)
	if !c.Incorporated.IsNew(hash, epoch, c.LiveBlock.CheckPoint) {
		return false
	}
	return c.LiveBlock.Validate(action)
}

func (c *Chain) NextBlock(epoch, checkpoint uint64, checkpointHash crypto.Hash, publisher crypto.Token) error {
	liveBlock, err := c.NewBlock(epoch, checkpoint, publisher)
	if err != nil {
		return err
	}
	c.LiveBlock = liveBlock
	return nil

}

func (c *Chain) SealOwnBlock() {
	c.LiveBlock.Seal(c.Credentials)
}

func (c *Chain) SealBlock(publishedAt time.Time, fees uint64, hash crypto.Hash, signature crypto.Signature) error {
	if c.LiveBlock == nil {
		return errors.New("no live block to be sealed")
	}
	c.LiveBlock.ProposedAt = publishedAt
	c.LiveBlock.Hash = hash
	c.LiveBlock.SealSignature = signature
	c.SealedBlocks[c.LiveBlock.Epoch] = c.LiveBlock
	c.LiveBlock = nil
	// todo check hash and signature?
	return nil
}

func (c *Chain) CommitOwnBlock() error {
	nextCommit := c.LastCommitEpoch + 1
	block, ok := c.SealedBlocks[nextCommit]
	if !ok {
		return errors.New("no sealed block")
	}
	validator := c.CommitState.Validator(nil, c.LastCommitEpoch)
	block.Revalidate(validator)
	c.CommitState.Incorporate(validator, block.Proposer)
	block.PreviousHash = c.LastCommitHash
	c.LastCommitEpoch += 1
	c.LastCommitHash = block.Hash
	delete(c.SealedBlocks, nextCommit-KeepLastN)
	return nil
}

func (c *Chain) CommitBlock(epoch uint64, blockhash crypto.Hash, previousblockhash crypto.Hash, invalidated []crypto.Hash) error {
	if epoch != c.LastCommitEpoch+1 {
		return errors.New("not a subsequent commit")
	}
	if c.LastCommitHash != previousblockhash {
		return errors.New("previous hash does not match")
	}
	block, ok := c.SealedBlocks[epoch]
	if !ok {
		return errors.New("could not find sealed block")
	}
	if !block.Hash.Equal(blockhash) {
		return errors.New("hash does not match")
	}
	exclude := make(map[crypto.Hash]struct{})
	for _, hash := range invalidated {
		exclude[hash] = struct{}{}
	}
	validator := c.CommitState.Validator(nil, c.LastCommitEpoch)
	for _, action := range block.Actions {
		hash := crypto.Hasher(action)
		if _, ok := exclude[hash]; !ok {
			validator.Validate(action)
		}
	}
	c.CommitState.Incorporate(validator, block.Proposer)
	block.PreviousHash = previousblockhash
	block.Invalidate = invalidated
	c.LastCommitEpoch += 1
	c.LastCommitHash = blockhash
	delete(c.SealedBlocks, epoch-KeepLastN)

	return nil
}

func (c *Chain) RolloverBlock(epoch uint64) error {
	if epoch < c.LastCommitEpoch {
		return errors.New("cannot rollbac before a commit epoch")
	}
	epochsAfter := make([]uint64, 0)
	found := false
	for blockEpoch, _ := range c.SealedBlocks {
		if blockEpoch > epoch {
			epochsAfter = append(epochsAfter, blockEpoch)
		} else if blockEpoch == epoch {
			found = true
		}
	}
	if !found {
		return errors.New("could not find block for the given epoch")
	}
	for _, blockEpoch := range epochsAfter {
		delete(c.SealedBlocks, blockEpoch)
	}
	if c.LiveBlock.Epoch > epoch {
		c.LiveBlock = nil
	}
	return nil
}

func (c *Chain) Shutdown() {
	c.CommitState.Shutdown()
}
