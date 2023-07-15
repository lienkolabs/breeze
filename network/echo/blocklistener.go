package echo

import (
	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network/trusted"
	"github.com/lienkolabs/breeze/protocol/chain"
)

// Block listener connects to a server providing blocks and channels every
// commited block.
type BlockListener struct {
	Connection *trusted.SignedConnection
	Block      chan *chain.Block
	shutdown   chan struct{}
	newBlock   *chain.Block
	sealed     map[uint64]*chain.Block
}

type BlockListenerConfig struct {
	Credentials         crypto.PrivateKey
	BlockServiceAddress string
	BlockServiveToken   crypto.Token
}

func NewBlockListener(config *BlockListenerConfig) (*BlockListener, error) {
	conn, err := trusted.Dial(config.BlockServiceAddress, config.Credentials, config.BlockServiveToken)

	listener := &BlockListener{
		Connection: conn,
		Block:      make(chan *chain.Block),
		shutdown:   make(chan struct{}),
		sealed:     make(map[uint64]*chain.Block),
	}
	if err != nil {
		return nil, err
	}
	messages := make(chan []byte)
	live := true

	go func() {
		msg := SubscribeProtocol{
			Code:      ProtocolCode{255, 255, 255, 255},
			FromEpoch: 0,
		}
		conn.Send(msg.Serialize())
		for {
			msg, err := conn.Read()
			if err != nil {
				if live {
					live = false
					listener.shutdown <- struct{}{}
				}
				return
			}
			messages <- msg
		}
	}()

	// close broken connections and process subscribe instructions
	go func() {
		for {
			select {
			case <-listener.shutdown:
				if live {
					listener.Connection.Shutdown()
				}
				live = false
				return
			case msg := <-messages:
				listener.NewMessage(msg)
			}
		}
	}()
	return listener, nil
}

func (l *BlockListener) Shutdown() {
	l.shutdown <- struct{}{}
}

func (l *BlockListener) NewMessage(msg []byte) {
	switch msg[0] {
	case blockcacheMsg:
		if len(msg) > 1 {
			block := chain.ParseBlock(msg[1:])
			if block != nil {
				l.Block <- block
			}
		}
	case actionMsg:
		if l.newBlock != nil {
			l.newBlock.Actions = append(l.newBlock.Actions, msg[1:])
		}
	case nextBlockMsg:
		if nextBlock := ParseBlockHeader(msg); nextBlock != nil {
			l.newBlock = &chain.Block{
				Epoch:          nextBlock.Epoch,
				CheckPoint:     nextBlock.Checkpoint,
				CheckpointHash: nextBlock.CheckpointHash,
				Proposer:       nextBlock.Publisher,
				Actions:        make([][]byte, 0),
			}
		}
	case sealBLockMsg:
		if seal := ParseBlockTail(msg); seal != nil {
			if l.newBlock != nil {
				l.newBlock.ProposedAt = seal.Timestamp
				l.newBlock.Hash = seal.Hash
				l.newBlock.SealSignature = seal.Signature
				l.sealed[l.newBlock.Epoch] = l.newBlock
				l.newBlock = nil
			}
		}
	case commitBlockMsg:
		if commit := ParseCommitBlock(msg); commit != nil {
			if sealed, ok := l.sealed[commit.Epoch]; ok && sealed.Hash.Equal(commit.Hash) {
				sealed.PreviousHash = commit.ParentHash
				sealed.Invalidate = commit.Invalidate
				l.Block <- sealed
				delete(l.sealed, commit.Epoch)
			}
		}
	case rolloverBlockMsg:
		if rollover := ParseRolloverBlock(msg); rollover != nil {
			for epoch := range l.sealed {
				if epoch > rollover.Epoch {
					delete(l.sealed, epoch)
				}
			}
		}
	}
}
