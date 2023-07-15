// Package poa implements the proof-of-authority consensus engine.
package poa

import (
	"fmt"
	"time"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network/echo"
	"github.com/lienkolabs/breeze/network/trusted"
	"github.com/lienkolabs/breeze/protocol/chain"
	"github.com/lienkolabs/breeze/protocol/state"
)

var blockInterval = time.Second

func NewProofOfAuthorityValidator(credentials crypto.PrivateKey, gatewayPort, broadcastPort int, walletPath string) error {
	actions := make(chan []byte)
	gateway, err := echo.NewActionsGateway(gatewayPort, credentials, trusted.AcceptAllConnections, actions)
	if err != nil {
		return err
	}
	pool, err := echo.NewBroadcastPool(credentials, trusted.AcceptAllConnections, broadcastPort)
	if err != nil {
		return err
	}
	blockstate := state.NewGenesisStateWithToken(credentials.PublicKey(), walletPath)
	epoch := uint64(0)
	ticker := time.NewTicker(blockInterval)
	block := &chain.Block{
		Epoch:          1,
		CheckPoint:     0,
		CheckpointHash: crypto.ZeroHash,
		Proposer:       credentials.PublicKey(),
		Actions:        make([][]byte, 0),
	}
	validator := blockstate.Validator(state.NewMutations(1), 1)
	go func() {
		for {
			select {
			case <-ticker.C:
				block.Seal(credentials)
				pool.BrodcastSealBlock(block.ProposedAt, block.Hash, block.SealSignature)
				pool.BrodcastCommitBlock(epoch, block.Hash)
				pool.Append(block)
				blockstate.Incorporate(validator, block.Proposer)
				hash := block.Hash
				epoch += 1
				validator = blockstate.Validator(state.NewMutations(epoch), epoch)
				fmt.Println(block.Epoch, len(block.Actions))
				block = block.NewBlock()
				pool.BrodcastNextBlock(epoch, epoch-1, hash, block.Proposer)
			case action := <-gateway.Actions:
				fmt.Println(action)
				if validator.Validate(action) {
					fmt.Println("incorporated")
					block.Actions = append(block.Actions, action)
					pool.BroadcastAction(action)
				}
			}
		}
	}()
	return nil
}
