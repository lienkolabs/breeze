package echo

import (
	"fmt"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network"
	"github.com/lienkolabs/breeze/network/trusted"
	"github.com/lienkolabs/breeze/protocol/chain"
)

// SocialNode is a general purpose network functionality for the deployment of
// specialized social protocols. It connects to a provider of blocks (either
// breeze original blocks or blocks filtered by other specific social protocols)
// filter acti....
type SocialNode struct {
	Credentials crypto.PrivateKey
	Chain       *chain.Chain
	State       chain.State
	shutdown    chan struct{}
	broadcast   *BroadcastPool
}

type SocialNodeConfig struct {
	Credentials            crypto.PrivateKey
	SocialCode             ProtocolCode
	BlockServiceAddress    string
	BlockServiveToken      crypto.Token
	BlockBroadcastPort     int
	BlockBroadcastFirewall network.ValidateConnection
}

func NewSocialNodeListener(config *SocialNodeConfig, state chain.State) (*SocialNode, error) {
	incomming, err := trusted.Dial(config.BlockServiceAddress, config.Credentials, config.BlockServiveToken)
	if err != nil {
		return nil, fmt.Errorf("could not connect to block service: %v", err)
	}
	node := &SocialNode{
		Credentials: config.Credentials,
		State:       state,
		shutdown:    make(chan struct{}),
	}
	node.broadcast, err = NewBroadcastPool(config.Credentials, config.BlockBroadcastFirewall, config.BlockBroadcastPort)
	if err != nil {
		return nil, fmt.Errorf("could not open block broadcast servide: %v", err)
	}

	messages := make(chan []byte)
	live := true

	go func() {
		// subscribe to receive actions filtered by the protocol code
		incomming.Send(append([]byte{subscribeMsg}, config.SocialCode[:]...))
		for {
			msg, err := incomming.Read()
			if err != nil {
				if live {
					live = false
					node.shutdown <- struct{}{}
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
			case <-node.shutdown:
				if live {
					incomming.Shutdown()
				}
				live = false
				return
			case msg := <-messages:
				node.NewMessage(msg)
			}
		}
	}()
	return node, nil
}

func (l *SocialNode) Shutdown() {
	l.shutdown <- struct{}{}
}

func (l *SocialNode) NewMessage(msg []byte) {
	switch msg[0] {
	case actionMsg:
		l.Chain.Validate(msg[1:])
	case nextBlockMsg:
		nextBlock := ParseBlockHeader(msg)
		l.Chain.NewBlock(nextBlock.Epoch, nextBlock.Checkpoint, l.Credentials.PublicKey())
	case sealBLockMsg:
		//seal := ParseBlockTail(msg)
		l.Chain.SealOwnBlock()
	case commitBlockMsg:
		//commit := ParseCommitBlock(msg)
		l.Chain.CommitOwnBlock()
	case rolloverBlockMsg:
		rollover := ParseRolloverBlock(msg)
		l.Chain.RolloverBlock(rollover.Epoch)
	}
}
