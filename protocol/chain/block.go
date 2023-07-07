package chain

import (
	"time"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/util"
)

type Block struct {
	Epoch         uint64
	CheckPoint    uint64
	Parent        crypto.Hash
	Publisher     crypto.Token
	PublishedAt   time.Time
	Actions       [][]byte
	Hash          crypto.Hash
	SealSignature crypto.Signature
	PreviousHash  crypto.Hash
	Invalidate    []crypto.Hash
	Validator     MutatingState
}

func (b *Block) NewBlock() *Block {
	return &Block{
		Epoch:      b.Epoch + 1,
		CheckPoint: b.Epoch + 1,
		Parent:     b.Hash,
		Publisher:  b.Publisher,
		Actions:    make([][]byte, 0),
	}
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
	util.PutUint64(b.Epoch, &bytes)
	util.PutUint64(b.CheckPoint, &bytes)
	util.PutHash(b.Parent, &bytes)
	util.PutToken(b.Publisher, &bytes)
	util.PutTime(b.PublishedAt, &bytes)
	util.PutUint32(uint32(len(b.Actions)), &bytes)
	for _, action := range b.Actions {
		util.PutByteArray(action, &bytes)
	}
	return bytes
}

func (b *Block) Seal(credentials crypto.PrivateKey) {
	b.PublishedAt = time.Now()
	b.Hash = crypto.Hasher(b.serializeForSeal())
	b.SealSignature = credentials.Sign(b.Hash[:])
}

func (b *Block) Serialize() []byte {
	bytes := b.serializeForSeal()
	util.PutSignature(b.SealSignature, &bytes)
	util.PutHash(b.Parent, &bytes)
	util.PutUint32(uint32(len(b.Invalidate)), &bytes)
	for _, hash := range b.Invalidate {
		util.PutHash(hash, &bytes)
	}
	return bytes
}

func ParseBlock(data []byte) *Block {
	position := 0
	block := Block{}
	block.Epoch, position = util.ParseUint64(data, position)
	block.CheckPoint, position = util.ParseUint64(data, position)
	block.Parent, position = util.ParseHash(data, position)
	block.Publisher, position = util.ParseToken(data, position)
	block.PublishedAt, position = util.ParseTime(data, position)
	block.Actions, position = util.ParseActionsArray(data, position)
	hash := crypto.Hasher(data)
	block.Hash, position = util.ParseHash(data, position)
	if !hash.Equal(block.Hash) {
		return nil
	}
	block.SealSignature, position = util.ParseSignature(data, position)
	if !block.Publisher.Verify(hash[:], block.SealSignature) {
		return nil
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
