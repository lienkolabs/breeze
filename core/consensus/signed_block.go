package consensus

import (
	"time"

	"github.com/lienkolabs/breeze/core/block"
	"github.com/lienkolabs/breeze/core/crypto"
	"github.com/lienkolabs/breeze/core/transactions"
)

type Signature struct {
	Hash      crypto.Hash
	Token     crypto.Token
	Signature crypto.Signature
}

type SignedBlock struct {
	Block      *block.Block
	Signatures []Signature
}

type SignedBlocks []*SignedBlock

func (blocks SignedBlocks) Less(i, j int) bool {
	return blocks[i].Block.Epoch() < blocks[j].Block.Epoch()
}

func (blocks SignedBlocks) Len() int {
	return len(blocks)
}

func (blocks SignedBlocks) Swap(i, j int) {
	blocks[i], blocks[j] = blocks[j], blocks[i]
}

type Blocks []*block.Block

func (blocks Blocks) Less(i, j int) bool {
	return blocks[i].Epoch() < blocks[j].Epoch()
}

func (blocks Blocks) Len() int {
	return len(blocks)
}

func (blocks Blocks) Swap(i, j int) {
	blocks[i], blocks[j] = blocks[j], blocks[i]
}

type Checkpoint struct {
	Validator       *block.MutatingState
	CheckpointEpoch uint64
	CheckpointHash  crypto.Hash
}

type processInstruction struct {
	instruction transactions.Transaction
	valid       chan bool
}

type instructionCache struct {
	instruction transactions.Transaction
	hash        crypto.Hash
}

func BlockBuilder(checkpoint *Checkpoint, epoch uint64, token crypto.PrivateKey, finish time.Time, pool *InstructionPool) chan *block.Block {
	build := block.NewBlock(checkpoint.CheckpointHash, checkpoint.CheckpointEpoch, epoch, token.PublicKey(), checkpoint.Validator)
	stop := time.NewTicker(time.Until(finish))
	communication := make(chan processInstruction)
	finished := make(chan *block.Block)
	running := true
	cache := make([]instructionCache, 0)
	go func() {
		for {
			select {
			case <-stop.C:
				finished <- build
				running = false
				for _, cached := range cache {
					pool.Queue(cached.instruction, cached.hash)
				}
				return
			case process := <-communication:
				process.valid <- build.Incorporate(process.instruction)
			}
		}
	}()

	go func() {
		valid := make(chan (bool))
		for {
			if !running {
				break
			}
			newInstruction, newHash := pool.Unqueue()
			if newInstruction != nil {
				cache = append(cache, instructionCache{newInstruction, newHash})
				communication <- processInstruction{
					instruction: newInstruction,
					valid:       valid,
				}
				if <-valid {
					cache = cache[0 : len(cache)-1]
				}
			}
		}
	}()
	return finished
}
