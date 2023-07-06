package echo

import (
	"time"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/util"
)

const (
	socialMsg byte = iota
	nextBlockMsg
	sealBLockMsg
	commitBlockMsg
	rolloverBlockMsg
	subscribeMsg
)

const protocolPos = 9

type ProtocolCode [4]byte

func validateCode(code ProtocolCode, data []byte) bool {
	for n := 0; n < 4; n++ {
		if code[n] != 255 && code[n] != data[protocolPos+n] {
			return false
		}
	}
	return true
}

type BlockHeader struct {
	Epoch          uint64
	Checkpoint     uint64
	CheckpointHash crypto.Hash
	Publisher      crypto.Token
}

func (b *BlockHeader) Serialize() []byte {
	data := []byte{nextBlockMsg}
	util.PutUint64(b.Epoch, &data)
	util.PutUint64(b.Checkpoint, &data)
	util.PutHash(b.CheckpointHash, &data)
	util.PutToken(b.Publisher, &data)

	return data
}

func ParseBlockHeader(data []byte) *BlockHeader {
	if len(data) == 0 {
		return nil
	}
	if data[0] != nextBlockMsg {
		return nil
	}
	position := 1
	var header BlockHeader
	header.Epoch, position = util.ParseUint64(data, position)
	header.Checkpoint, position = util.ParseUint64(data, position)
	header.CheckpointHash, position = util.ParseHash(data, position)
	header.Publisher, position = util.ParseToken(data, position)
	if position != len(data) {
		return nil
	}
	return &header
}

type BlockTail struct {
	Timestamp time.Time
	Hash      crypto.Hash
	Signature crypto.Signature
}

func (b *BlockTail) Serialize() []byte {
	data := []byte{sealBLockMsg}
	util.PutTime(b.Timestamp, &data)
	util.PutHash(b.Hash, &data)
	util.PutSignature(b.Signature, &data)
	return data
}

func ParseBlockTail(data []byte) *BlockTail {
	if len(data) == 0 {
		return nil
	}
	if data[0] != sealBLockMsg {
		return nil
	}
	position := 1
	var tail BlockTail
	tail.Timestamp, position = util.ParseTime(data, position)
	tail.Hash, position = util.ParseHash(data, position)
	tail.Signature, position = util.ParseSignature(data, position)
	if position != len(data) {
		return nil
	}
	return &tail
}

type CommitBlock struct {
	Epoch      uint64
	Hash       crypto.Hash
	ParentHash crypto.Hash
	Invalidate []crypto.Hash
}

func (b *CommitBlock) Serialize() []byte {
	data := []byte{commitBlockMsg}
	util.PutUint64(b.Epoch, &data)
	util.PutHash(b.Hash, &data)
	util.PutHash(b.ParentHash, &data)
	util.PutHashArray(b.Invalidate, &data)
	return data
}

func ParseCommitBlock(data []byte) *CommitBlock {
	if len(data) == 0 {
		return nil
	}
	if data[0] != commitBlockMsg {
		return nil
	}
	position := 1
	var commit CommitBlock
	commit.Epoch, position = util.ParseUint64(data, position)
	commit.Hash, position = util.ParseHash(data, position)
	commit.ParentHash, position = util.ParseHash(data, position)
	commit.Invalidate, position = util.ParseHashArray(data, position)
	if position != len(data) {
		return nil
	}
	return &commit
}

type RolloverBlock struct {
	Epoch uint64
}

func (b *RolloverBlock) Serialize() []byte {
	data := []byte{rolloverBlockMsg}
	util.PutUint64(b.Epoch, &data)
	return data
}

func ParseRolloverBlock(data []byte) *RolloverBlock {
	if len(data) == 0 {
		return nil
	}
	if data[0] != rolloverBlockMsg {
		return nil
	}
	position := 1
	var rollover RolloverBlock
	rollover.Epoch, position = util.ParseUint64(data, position)
	if position != len(data) {
		return nil
	}
	return &rollover
}
