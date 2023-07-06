// Package poa implements the proof-of-authority consensus engine.
package poa

import (
	"time"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network/echo"
	"github.com/lienkolabs/breeze/network/trusted"
	"github.com/lienkolabs/breeze/protocol/chain"
	"github.com/lienkolabs/breeze/protocol/state"
)

const ActionsGatewayPort = 3100
const BroadcastPoolPort = 3101

var blockInterval = time.Second

func NewProofOfAuthorityValidator(credentials crypto.PrivateKey) error {
	actions := make(chan []byte)
	gateway, err := echo.NewActionsGateway(ActionsGatewayPort, credentials, trusted.AcceptAllConnections, actions)
	if err != nil {
		return err
	}
	pool, err := echo.NewBroadcastPool(credentials, trusted.AcceptAllConnections, BroadcastPoolPort)
	if err != nil {
		return err
	}
	blockstate := state.NewGenesisStateWithToken(credentials.PublicKey())
	epoch := uint64(0)
	ticker := time.NewTicker(blockInterval)
	block := &chain.Block{
		Epoch:      1,
		CheckPoint: 0,
		Parent:     crypto.ZeroHash,
		Publisher:  credentials.PublicKey(),
		Actions:    make([][]byte, 0),
	}
	validator := blockstate.Validator(state.NewMutations(), 1)
	go func() {
		for {
			select {
			case <-ticker.C:
				block.Seal(credentials)
				pool.BrodcastSealBlock(block.PublishedAt, block.Hash, block.SealSignature)
				pool.BrodcastCommitBlock(epoch, block.Hash)
				blockstate.Incorporate(validator)
				hash := block.Hash
				epoch += 1
				validator = blockstate.Validator(state.NewMutations(), epoch)
				block = block.NewBlock()
				pool.BrodcastNextBlock(epoch, epoch-1, hash, block.Publisher)
			case action := <-gateway.Actions:
				if validator.Validate(action) {
					block.Actions = append(block.Actions, action)
					pool.BroadcastAction(action)
				}
			}
		}
	}()
	return nil
}
