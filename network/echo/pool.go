package echo

import (
	"sync"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/protocol/actions"
)

type ActionPool struct {
	queue   []crypto.Hash // order in which instructions are received
	actions map[crypto.Hash]actions.Action
	mu      sync.Mutex
}

func NewActionPool() *ActionPool {
	return &ActionPool{
		queue:   make([]crypto.Hash, 0),
		actions: make(map[crypto.Hash]actions.Action),
	}
}

func (pool *ActionPool) Unqueue() (actions.Action, crypto.Hash) {
	if len(pool.queue) == 0 {
		return nil, crypto.ZeroHash
	}
	pool.mu.Lock()
	defer pool.mu.Unlock()
	for n, hash := range pool.queue {
		if action, ok := pool.actions[hash]; ok {
			pool.queue = pool.queue[n+1:]
			delete(pool.actions, hash)
			return action, hash
		}
	}
	pool.queue = pool.queue[:0]
	return nil, crypto.ZeroHash
}

func (pool *ActionPool) Queue(action actions.Action, hash crypto.Hash) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	pool.queue = append(pool.queue, hash)
	pool.actions[hash] = action
}

func (pool *ActionPool) Delete(hash crypto.Hash) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	delete(pool.actions, hash)
}

func (pool *ActionPool) DeleteArray(hashes []crypto.Hash) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	for _, hash := range hashes {
		delete(pool.actions, hash)
	}
}
