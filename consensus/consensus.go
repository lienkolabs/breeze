package consensus

import (
	"github.com/lienkolabs/breeze/crypto"
)

type Network interface {
	CommitBlock(uint64)
	RolloverBlock(uint64)
	NextBlock(epoch, checkpoint uint64, parent crypto.Hash)
	SealBlock()
	Validated([]byte)
	Gateway() chan []byte
}
