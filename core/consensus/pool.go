package consensus

import (
	"sync"

	"github.com/lienkolabs/breeze/core/crypto"
	"github.com/lienkolabs/breeze/core/transactions"
)

type InstructionPool struct {
	queue        []crypto.Hash // order in which instructions are received
	instructions map[crypto.Hash]transactions.Transaction
	mu           sync.Mutex
}

func NewInstructionPool() *InstructionPool {
	return &InstructionPool{
		queue:        make([]crypto.Hash, 0),
		instructions: make(map[crypto.Hash]transactions.Transaction),
	}
}

func (pool *InstructionPool) Unqueue() (transactions.Transaction, crypto.Hash) {
	if len(pool.queue) == 0 {
		return nil, crypto.ZeroHash
	}
	pool.mu.Lock()
	defer pool.mu.Unlock()
	for n, hash := range pool.queue {
		if instruction, ok := pool.instructions[hash]; ok {
			pool.queue = pool.queue[n+1:]
			delete(pool.instructions, hash)
			return instruction, hash
		}
	}
	pool.queue = pool.queue[:0]
	return nil, crypto.ZeroHash
}

func (pool *InstructionPool) Queue(instruction transactions.Transaction, hash crypto.Hash) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	pool.queue = append(pool.queue, hash)
	pool.instructions[hash] = instruction
}

func (pool *InstructionPool) Delete(hash crypto.Hash) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	delete(pool.instructions, hash)
}

func (pool *InstructionPool) DeleteArray(hashes []crypto.Hash) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	for _, hash := range hashes {
		delete(pool.instructions, hash)
	}
}
