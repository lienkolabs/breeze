package echo

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network"
	"github.com/lienkolabs/breeze/network/trusted"
)

type listenerConnection struct {
	conn *trusted.SignedConnection
	code ProtocolCode
}

type BroadcastPool struct {
	mu   sync.Mutex
	conn map[crypto.Token]*listenerConnection
}

func NewBroadcastPool(credentials crypto.PrivateKey, validator network.ValidateConnection, port int) (*BroadcastPool, error) {
	pool := &BroadcastPool{
		mu:   sync.Mutex{},
		conn: make(map[crypto.Token]*listenerConnection),
	}

	listeners, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		return nil, err
	}

	messages := make(chan trusted.Message)
	shutdown := make(chan crypto.Token)

	// accept new connecttions
	go func() {
		for {
			if conn, err := listeners.Accept(); err == nil {
				tConn, err := trusted.PromoteConnection(conn, credentials, validator)
				if err != nil {
					conn.Close()
				} else {
					pool.mu.Lock()
					pool.conn[tConn.Token] = &listenerConnection{
						conn: tConn,
					}
					pool.mu.Unlock()
					tConn.Listen(messages, shutdown)
				}
			} else {
				return
			}
		}
	}()

	// close broken connections and process subscribe instructions
	go func() {
		for {
			select {
			case token := <-shutdown:
				pool.mu.Lock()
				if listener, ok := pool.conn[token]; ok {
					listener.conn.Shutdown()
				}
				delete(pool.conn, token)
				pool.mu.Unlock()
			case msg := <-messages:
				if msg.Data[0] == subscribeMsg {
					if trusted, ok := pool.conn[msg.Token]; ok && len(msg.Data) == 5 {
						for n := 0; n < 4; n++ {
							trusted.code[n] = msg.Data[n+1]
						}
					}
				}
			}
		}
	}()
	return pool, nil
}

func (pool *BroadcastPool) BroadcastAction(data []byte) {
	msg := []byte{socialMsg}
	msg = append(msg, data...)
	pool.Broadcast(msg)
}

func (pool *BroadcastPool) BroadcastRollover(epoch uint64) {
	rollover := RolloverBlock{
		Epoch: epoch,
	}
	msg := rollover.Serialize()
	pool.Broadcast(msg)
}

func (pool *BroadcastPool) BrodcastSealBlock(timestamp time.Time, fees uint64, hash crypto.Hash, signature crypto.Signature) {
	seal := BlockTail{
		Timestamp:     timestamp,
		FeesCollected: fees,
		Hash:          hash,
		Signature:     signature,
	}
	msg := seal.Serialize()
	pool.Broadcast(msg)
}

func (pool *BroadcastPool) BrodcastNextBlock(epoch, checkpoint uint64, publisher crypto.Token) {
	nextBlock := BlockHeader{
		Epoch:      epoch,
		Checkpoint: checkpoint,
		Publisher:  publisher,
	}
	msg := nextBlock.Serialize()
	pool.Broadcast(msg)
}

func (pool *BroadcastPool) BrodcastCommitBlock(epoch uint64, hash crypto.Hash) {
	commit := CommitBlock{
		Epoch: epoch,
		Hash:  hash,
	}
	msg := commit.Serialize()
	pool.Broadcast(msg)
}

/*socialMsg byte = iota
nextBlockMsg
sealBLockMsg
commitBlockMsg
rolloverBlockMsg
subscribeMsg
*/

func (pool *BroadcastPool) Broadcast(data []byte) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	for _, listener := range pool.conn {
		if validateCode(listener.code, data) {
			listener.conn.Send(data)
		}
	}
}
