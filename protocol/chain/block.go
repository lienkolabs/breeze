package chain

import (
	"time"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/protocol"
	"github.com/lienkolabs/breeze/util"
)

// Block groups actions sharing a common timestamp defined by epoch.
// Blocks are proposed against a checkpoint epoch prior to proposed epoch
// The checkpoint is represented by a hash (parent block).
// The proposer is the one gathering actions and grouping it.
// Publisher is any listener of blocks that filters down actions in terms of a
// specialized protocol.
// Hash
type Block struct {
	Protocol         protocol.Code // 0 for breeze protocol
	Epoch            uint64
	CheckPoint       uint64
	CheckpointHash   crypto.Hash
	Proposer         crypto.Token
	Publisher        crypto.Token // Only if prototol is not breeze
	ProposedAt       time.Time
	Actions          [][]byte
	Hash             crypto.Hash
	SealSignature    crypto.Signature
	PublishHash      crypto.Hash      // Only if protocol is not breeze
	PublishSignature crypto.Signature // Only if protocol is not breeze
	PreviousHash     crypto.Hash      // Hash of the recognzied prior sequence of blocks
	Invalidate       []crypto.Hash
	Validator        MutatingState
}

func (b *Block) NewBlock() *Block {
	return &Block{
		Protocol:       b.Protocol,
		Epoch:          b.Epoch + 1,
		CheckPoint:     b.Epoch + 1,
		CheckpointHash: b.Hash,
		Proposer:       b.Proposer,
		Actions:        make([][]byte, 0),
	}
}

func (b *Block) Header() []byte {
	bytes := make([]byte, 0)
	util.PutUint32(uint32(b.Protocol), &bytes)
	util.PutUint64(b.Epoch, &bytes)
	util.PutUint64(b.CheckPoint, &bytes)
	util.PutHash(b.CheckpointHash, &bytes)
	util.PutToken(b.Proposer, &bytes)
	if b.Protocol != 0 {
		util.PutToken(b.Publisher, &bytes)
	}
	util.PutTime(b.ProposedAt, &bytes)
	util.PutUint32(uint32(len(b.Actions)), &bytes)
	return bytes
}

func (b *Block) Tail() []byte {
	bytes := make([]byte, 0)
	util.PutSignature(b.SealSignature, &bytes)
	util.PutHash(b.CheckpointHash, &bytes)
	util.PutUint32(uint32(len(b.Invalidate)), &bytes)
	for _, hash := range b.Invalidate {
		util.PutHash(hash, &bytes)
	}
	return bytes
}

func (b *Block) Validate(action []byte) bool {
	if !b.Validator.Validate(action) {
		return false
	}
	b.Actions = append(b.Actions, action)
	return true
}

func (b *Block) serializeForSeal() []byte {
	bytes := make([]byte, 0)
	util.PutUint32(uint32(b.Protocol), &bytes)
	util.PutUint64(b.Epoch, &bytes)
	util.PutUint64(b.CheckPoint, &bytes)
	util.PutHash(b.CheckpointHash, &bytes)
	util.PutToken(b.Proposer, &bytes)
	if b.Protocol != 0 {
		util.PutToken(b.Publisher, &bytes)
	}
	util.PutTime(b.ProposedAt, &bytes)
	util.PutActionsArray(b.Actions, &bytes)
	return bytes
}

func (b *Block) Seal(credentials crypto.PrivateKey) {
	b.ProposedAt = time.Now()
	data := b.serializeForSeal()
	b.Hash = crypto.Hasher(data)
	b.SealSignature = credentials.Sign(b.Hash[:])
}

func (b *Block) serializeForPublish() []byte {
	bytes := b.serializeForSeal()
	util.PutHash(b.Hash, &bytes)
	util.PutSignature(b.SealSignature, &bytes)
	return bytes
}

func (b *Block) Publish(credentials crypto.PrivateKey) {
	b.PublishHash = crypto.Hasher(b.serializeForPublish())
	b.PublishSignature = credentials.Sign(b.PublishHash[:])
}

func (b *Block) Serialize() []byte {
	bytes := b.serializeForPublish()
	if b.Protocol != 0 {
		util.PutHash(b.PublishHash, &bytes)
		util.PutSignature(b.PublishSignature, &bytes)
	}
	util.PutHash(b.PreviousHash, &bytes)
	util.PutHashArray(b.Invalidate, &bytes)
	return bytes
}

func ParseBlock(data []byte) *Block {
	if data[0] != 0 {
		return nil
	}
	position := 0
	block := Block{}
	var protocolNum uint32
	protocolNum, position = util.ParseUint32(data, position)
	block.Protocol = protocol.Code(protocolNum)
	block.Epoch, position = util.ParseUint64(data, position)
	block.CheckPoint, position = util.ParseUint64(data, position)
	block.CheckpointHash, position = util.ParseHash(data, position)
	block.Proposer, position = util.ParseToken(data, position)
	if block.Protocol != 0 {
		block.Publisher, position = util.ParseToken(data, position)
	}
	block.ProposedAt, position = util.ParseTime(data, position)
	block.Actions, position = util.ParseActionsArray(data, position)
	hash := crypto.Hasher(data[0:position])
	block.Hash, position = util.ParseHash(data, position)
	if block.Protocol == 0 && !hash.Equal(block.Hash) {
		return nil
	}
	block.SealSignature, position = util.ParseSignature(data, position)
	if !block.Proposer.Verify(hash[:], block.SealSignature) {
		return nil
	}
	if block.Protocol != 0 {
		block.PublishHash, position = util.ParseHash(data, position)
		block.PublishSignature, position = util.ParseSignature(data, position)
		if block.Publisher.Verify(block.PublishHash[:], block.PublishSignature) {
			return nil
		}
	}
	block.PreviousHash, position = util.ParseHash(data, position)
	block.Invalidate, _ = util.ParseHashArray(data, position)
	return &block
}

func (b *Block) Revalidate(v MutatingState) {
	b.Invalidate = make([]crypto.Hash, 0)
	for _, action := range b.Actions {
		if !v.Validate(action) {
			hash := crypto.Hasher(action)
			b.Invalidate = append(b.Invalidate, hash)
		}
	}
	b.Validator = v
}
