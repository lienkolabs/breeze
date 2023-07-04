// Package poa implements the proof-of-authority consensus engine.
package poa

import (
	"time"

	"github.com/lienkolabs/breeze/protocol"
	"github.com/lienkolabs/breeze/protocol/actions"
)

var blockInterval = time.Second

func NewProofOfAuthorityValidator(network *Node, node *protocol.ValidatorNode) error {
	block := node.NextBlock(0, 1)
	epoch := uint64(0)
	ticker := time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				block.Seal(node.Credentials)
				network.broadcast.BrodcastSealBlock(block.PublishedAt, block.FeesCollected, block.Hash, block.Signature)
				node.Commit(block.Epoch(), block.Hash)
				network.broadcast.BrodcastCommitBlock(block.Epoch(), block.Hash)
				epoch += 1
				block = node.NextBlock(epoch, epoch-1)
				network.broadcast.BrodcastNextBlock(block.Epoch(), block.CheckPoint, block.Publisher)
			case msg := <-network.gateway.Actions:
				action := actions.ParseAction(msg)
				if action != nil && block.Incorporate(action) {
					network.broadcast.BroadcastAction(msg)
				}
			}
		}
	}()
	return nil
}
