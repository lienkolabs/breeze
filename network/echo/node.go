package echo

import (
	"errors"
	"time"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network"
)

// this is a no ambiguity block interface. only one block will be provided for
// each epoch, and only one block will be live for validation at each instant.
// commit can be delayed nonetheless
type ProtocolValidator interface {
	Validate(action []byte) bool
	NextBlock(epoch, checkpoint uint64, checkpointHash crypto.Hash, publisher crypto.Token) error
	SealBlock(publishedAt time.Time, hash crypto.Hash, signature crypto.Signature) error
	CommitBlock(epoch uint64, blockhash crypto.Hash, previousblockhash crypto.Hash, invalidated []crypto.Hash) error
	RolloverBlock(epoch uint64) error
	Shutdown()
}

type ProtocolNode struct {
	Validator       ProtocolValidator
	IncomingAddress string
	IncomingToken   crypto.Token
	OutcomingPort   int
	Credentials     crypto.PrivateKey
	Firewall        network.ValidateConnection
	listener        *Listener
	broadcast       *BroadcastPool
	shutdown        chan struct{}
}

func (p *ProtocolNode) Shutdown() {
	p.shutdown <- struct{}{}
}

func (p *ProtocolNode) Start() error {
	if p.Validator == nil {
		return errors.New("protocol node must have a valid protocol validator")
	}
	var err error
	p.listener, err = NewListener(p.Credentials, p.IncomingAddress, p.IncomingToken)
	if err != nil {
		return err
	}
	p.broadcast, err = NewBroadcastPool(p.Credentials, p.Firewall, p.OutcomingPort)
	if err != nil {
		return err
	}
	p.shutdown = make(chan struct{})

	go func() {
		for {
			select {
			case msg := <-p.listener.Incoming:
				p.incorporate(msg)
			case <-p.shutdown:
				p.listener.Shutdown()
				p.broadcast.Shutdown()
				return
			}
		}

	}()
	return nil
}

func (node *ProtocolNode) incorporate(msg []byte) bool {
	if len(msg) == 0 {
		return false
	}
	if msg[0] == commitBlockMsg || msg[0] == rolloverBlockMsg || msg[0] == nextBlockMsg || msg[0] == sealBLockMsg {
		if len(msg) != 9 {
			return false
		}
		switch msg[0] {
		case nextBlockMsg:
			header := ParseBlockHeader(msg)
			if header == nil {
				return false
			}
			node.Validator.NextBlock(header.Epoch, header.Checkpoint, header.CheckpointHash, header.Publisher)
		case commitBlockMsg:
			commit := ParseCommitBlock(msg)
			if commit == nil {
				return false
			}
			node.Validator.CommitBlock(commit.Epoch, commit.Hash, commit.ParentHash, commit.Invalidate)
		case rolloverBlockMsg:
			rollover := ParseRolloverBlock(msg)
			if rollover == nil {
				return false
			}
			node.Validator.RolloverBlock(rollover.Epoch)
		case sealBLockMsg:
			tail := ParseBlockTail(msg)
			if tail == nil {
				return false
			}
			node.Validator.SealBlock(tail.Timestamp, tail.Hash, tail.Signature)
		}
		node.broadcast.Broadcast(msg)
		return true
	}
	if msg[0] != socialMsg {
		return false
	}
	if !node.Validator.Validate(msg[1:]) {
		return false
	}
	node.broadcast.Broadcast(msg)
	return true
}
