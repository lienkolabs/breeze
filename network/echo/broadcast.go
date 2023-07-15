package echo

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network"
	"github.com/lienkolabs/breeze/network/trusted"
	"github.com/lienkolabs/breeze/protocol/chain"
)

const MaxCacheSize = 60 * 15

type cache struct {
	mu         sync.Mutex
	firstEpoch uint64
	lastEpoch  uint64
	blocks     map[uint64]*chain.Block
}

func newCache() *cache {
	return &cache{
		mu:     sync.Mutex{},
		blocks: make(map[uint64]*chain.Block),
	}
}

// Append add a new block into cache. It pressuposes append will be sequential
// meaning the append block is on epoch of c.lastEpoch + 1. It does not check.
// It pressuposes that cache will be called with the pool lock. It cannot be
// used in other contexts.
func (c *cache) Append(block *chain.Block) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.blocks[block.Epoch] = block
	delete(c.blocks, block.Epoch-MaxCacheSize)
	if block.Epoch > MaxCacheSize {
		c.firstEpoch += 1
	}
	c.lastEpoch = block.Epoch
}

// Get copy of cache pointers starting from start epoch. It pressuposes that
// cache will be called with the pool lock. It cannot be used in other contexts.
func (c *cache) GetCopy(epochs ...uint64) []*chain.Block {
	c.mu.Lock()
	defer c.mu.Unlock()
	var start, end uint64
	if len(epochs) == 1 {
		start = epochs[0]
		end = c.lastEpoch
	} else if len(epochs) == 2 {
		start = epochs[0]
		end = epochs[1]
	} else {
		log.Printf("unexpected call on echo.cache.GetCopy: %v", epochs)
		return nil
	}
	if start > end {
		return nil
	}
	if start < c.firstEpoch {
		start = c.firstEpoch
	}
	all := make([]*chain.Block, 0, end-start+1)
	for epoch := start; epoch <= end; epoch++ {
		if block, ok := c.blocks[epoch]; ok {
			all = append(all, block)
		}
	}
	return all
}

type listenerConnection struct {
	conn          *trusted.SignedConnection
	live          bool
	code          ProtocolCode
	firstnewblock uint64 // to be set as true when the first new block message is incorporated
	all           bool
	append        chan []byte
	shutdown      chan []byte
}

func (l *listenerConnection) Send(data []byte) {
	fmt.Println("send")
	l.append <- data
	fmt.Println("sent")
}

func (l *listenerConnection) SendAction(action []byte) {
	if l.all || validateCode(l.code, action) {
		l.append <- action
	}
}

type listenQueue struct {
	mu    sync.Mutex
	cache [][]byte
}

func newQueue() *listenQueue {
	return &listenQueue{
		mu:    sync.Mutex{},
		cache: make([][]byte, 0),
	}
}

func (l *listenQueue) append(data []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cache = append(l.cache, data)
}

func (l *listenQueue) pop() []byte {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.cache) == 0 {
		return nil
	}
	item := l.cache[0]
	l.cache = l.cache[1:]
	return item
}

func (l *listenQueue) empty() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.cache) == 0
}

func (l *listenerConnection) Subscribe(subscribe *SubscribeProtocol, pool *BroadcastPool) {

	// subscribe is syncrhnous with pool select... cache will not be updated
	// until this function returns
	l.live = true
	syncJob := pool.cache.GetCopy(subscribe.FromEpoch)

	// open channels to communicate with processes and set subscription details
	l.append = make(chan []byte)
	l.shutdown = make(chan []byte)
	l.code = subscribe.Code
	l.all = (subscribe.Code[0] & subscribe.Code[1] & subscribe.Code[2] & subscribe.Code[3]) == 255

	// create cache for new messages while syncrhonization requests are ongoing
	// or breeze activity temporarily exceeds network capacity
	// TODO: create logic to limit max cache size
	cache := newQueue()

	// channel used to for internal communication
	send := make(chan []byte)

	// state of the pool
	closed := false // to be set as true on shutdown request
	synced := true  // to be set as false if there is cache blocks to be sync

	if len(syncJob) > 0 {
		synced = false
		// block synchronization job. New messages will be cached until this job
		// is finished.
		go func() {
			lastblock := uint64(0)
			for {
				if len(syncJob) == 0 {
					if l.firstnewblock > lastblock+1 {
						// loop to wait to new blocks on cache... will wait 3
						// seconds at maximum.
						count := 0
						for {
							if pool.cache.lastEpoch >= l.firstnewblock-1 {
								syncJob = pool.cache.GetCopy(lastblock+1, l.firstnewblock-1)
								break
							} else {
								// TODO
							}
							if count > 3 {
								// giveup and continue
								l.firstnewblock = lastblock
								break
							}
							time.Sleep(time.Second)
							count += 1
						}
					} else {
						break
					}
				}
				block := syncJob[0]
				lastblock = block.Epoch
				syncJob = syncJob[1:]
				var bytes []byte
				if l.all {
					bytes = NewBlockCache(block)
				} else {
					bytes = NewFilteredBlockCache(block, l.code)
				}
				l.conn.Send(bytes)
			}
			synced = true
			send <- nil // force cache loop
		}()
	}

	go func() {
		for {
			select {
			case data := <-l.append:
				available := (!closed) && synced && cache.empty()
				if available {
					send <- data
				} else {
					cache.append(data)
				}
			case <-l.shutdown:
				close(send)
				close(l.append)
				closed = true
				return
			}
		}
	}()

	// send loop
	go func() {
		for {
			// empty cache
			for {
				if cache.empty() {
					break
				}
				if pop := cache.pop(); pop != nil {
					l.conn.Send(pop)
				}
			}
			// keep listening until new send request
			data := <-send
			if len(data) > 0 {
				l.conn.Send(data)
			} else if closed {
				// shutdown was called
				return
			}
		}
	}()
}

// BroacastPool provides a service of sending information about new blocks to
// interested parties. Clients connecting to this service must send a subscribe
// message informing the social protocol it wants to receive information about.
// Blocks with filtered actions will be sent to clients.
// Broadcast pool keeps a cache of 15 minutes worth of blocks that can be used
// to synchronize client with recent blocks.
// BroadcastPool can be used either by validator nodes in the breeze network to
// inform peers and listeners to block formation, or can be used by other middle
// ware or specialized protocol nodes within breeze architecture.
type BroadcastPool struct {
	conn            map[crypto.Token]*listenerConnection
	cache           *cache
	broadcast       chan []byte
	broadcastAction chan []byte
	nextBlock       chan uint64
	block           chan *chain.Block
}

// NewBroadcastPool instances a new pool listening to connections on the
// provided. Connections are signed naked connections. Validator specificies if
// a signed connection associated to given token is allowed to be incorporated
// into the pool. use network.AcceptAllConnections to provide an open interface.
// Credentials are used to establish connection and also to publish filtered
// blocks. Unfiletred blocks are passed with original signature.
func NewBroadcastPool(credentials crypto.PrivateKey, validator network.ValidateConnection, port int) (*BroadcastPool, error) {
	pool := &BroadcastPool{
		conn:            make(map[crypto.Token]*listenerConnection),
		cache:           newCache(),
		broadcast:       make(chan []byte),
		broadcastAction: make(chan []byte),
		nextBlock:       make(chan uint64),
		block:           make(chan *chain.Block),
	}

	listeners, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		return nil, err
	}

	messages := make(chan trusted.Message)
	shutdown := make(chan crypto.Token)
	incoming := make(chan *listenerConnection)

	// accept new connecttions
	go func() {
		for {
			if conn, err := listeners.Accept(); err == nil {
				tConn, err := trusted.PromoteConnection(conn, credentials, validator)
				if err != nil {
					conn.Close()
				} else {
					incoming <- &listenerConnection{conn: tConn}

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
			case msg := <-pool.broadcast:
				for _, listener := range pool.conn {
					listener.Send(msg)
				}
			case msg := <-pool.broadcastAction:
				for _, listener := range pool.conn {
					listener.SendAction(msg)
				}
			case epoch := <-pool.nextBlock:
				for _, listener := range pool.conn {
					if listener.firstnewblock == 0 {
						listener.firstnewblock = epoch
					}
				}
			case block := <-pool.block:
				pool.cache.Append(block)
			case listener := <-incoming:
				pool.conn[listener.conn.Token] = listener
				listener.conn.Listen(messages, shutdown)
			case token := <-shutdown:
				if listener, ok := pool.conn[token]; ok {
					listener.conn.Shutdown()
					delete(pool.conn, token)
				}
				delete(pool.conn, token)
			case msg := <-messages:
				if msg.Data[0] == subscribeMsg {
					subscribe := ParseSubscribeProtocol(msg.Data)
					if listener, ok := pool.conn[msg.Token]; ok {
						listener.Subscribe(subscribe, pool)
					}
				}
			}
		}
	}()
	return pool, nil
}

// Instruct shutdown without response. TODO: implemente context logic.
func (pool *BroadcastPool) Shutdown() {
	for _, conn := range pool.conn {
		conn.conn.Shutdown()
	}
}

// Broadcast action to all connected parties subscribing to protocol codes
// compatible with the action.
func (pool *BroadcastPool) BroadcastAction(data []byte) {
	msg := []byte{actionMsg}
	msg = append(msg, data...)
	pool.broadcastAction <- msg
}

// Broadcast message to rollover blockchain to specified epoch.
func (pool *BroadcastPool) BroadcastRollover(epoch uint64) {
	rollover := RolloverBlock{
		Epoch: epoch,
	}
	msg := rollover.Serialize()
	pool.Broadcast(msg)
}

// Broadcast message with details about block sealing event.
func (pool *BroadcastPool) BrodcastSealBlock(timestamp time.Time, hash crypto.Hash, signature crypto.Signature) {
	seal := BlockTail{
		Timestamp: timestamp,
		Hash:      hash,
		Signature: signature,
	}
	msg := seal.Serialize()
	pool.Broadcast(msg)
}

// Broadcast message to start formation of new block. It will mark as the first
// full block cycle new subscriptions so that synchronization can run smoothly.
func (pool *BroadcastPool) BrodcastNextBlock(epoch, checkpoint uint64, checkpointHash crypto.Hash, publisher crypto.Token) {
	nextBlock := BlockHeader{
		Epoch:          epoch,
		Checkpoint:     checkpoint,
		CheckpointHash: checkpointHash,
		Publisher:      publisher,
	}
	msg := nextBlock.Serialize()
	pool.nextBlock <- epoch
	pool.Broadcast(msg)
}

// Broadcast message to consider the sealed block for given eposh and given
// hash commited. Commited blocks can only be rolled over on disaster recovery
// through swell checkpoint mechanism.
func (pool *BroadcastPool) BrodcastCommitBlock(epoch uint64, hash crypto.Hash) {
	commit := CommitBlock{
		Epoch: epoch,
		Hash:  hash,
	}
	msg := commit.Serialize()
	pool.Broadcast(msg)
}

// Broadcast messages to all connected parties. To broadcast action use
// BroadcastAction method that implements protocol code filtering.
func (pool *BroadcastPool) Broadcast(data []byte) {
	pool.broadcast <- data
}

// Instruct the pool to append a new block to cache.
func (pool *BroadcastPool) Append(block *chain.Block) {
	pool.block <- block
}
