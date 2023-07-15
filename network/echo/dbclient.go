package echo

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network/trusted"
	"github.com/lienkolabs/breeze/util"
)

type Queue struct {
	pool     [][]byte
	get      chan chan []byte
	set      chan []byte
	waiting  chan []byte
	shutdown chan struct{}
}

func (q *Queue) Add(item []byte) {
	q.set <- item
}

func (q *Queue) Get(response chan []byte) {
	q.get <- response
}

func (q *Queue) Shutdown() {
	q.shutdown <- struct{}{}
}

func NewQueue() *Queue {
	queue := &Queue{
		pool:     make([][]byte, 0),
		get:      make(chan chan []byte),
		set:      make(chan []byte),
		shutdown: make(chan struct{}),
	}
	live := true
	go func() {
		for {
			select {
			case resp := <-queue.get:
				if len(queue.pool) > 0 {
					resp <- queue.pool[0]
					queue.pool = queue.pool[1:]
				} else if !live {
					resp <- []byte{}
					return
				} else if queue.waiting == nil {
					queue.waiting = resp
				} else {
					// TODO: when you get here?
					close(resp)
				}
			case item := <-queue.set:
				if queue.waiting != nil {
					queue.waiting <- item
					queue.waiting = nil
				} else {
					queue.pool = append(queue.pool, item)
				}
			case <-queue.shutdown:
				if queue.waiting != nil {
					queue.waiting <- []byte{}
					return
				}
				live = false
			}
		}
	}()
	return queue
}

type DBClient struct {
	mu     sync.Mutex
	Epoch  uint64
	Conn   *trusted.SignedConnection
	Queue  *Queue
	Finish uint64
	Live   bool
}

func (db *DBClient) Shutdown() {
	db.Queue.Shutdown()
	db.Conn.Shutdown()
}

type DBClientConfig struct {
	ProviderAddress string
	ProviderToken   crypto.Token
	Credentials     crypto.PrivateKey
	KeepAlive       bool
}

func (db *DBClient) Receive(resp chan []byte) {
	db.Queue.Get(resp)
}

func NewDBClient(config DBClientConfig) (*DBClient, error) {
	text, _ := json.Marshal(config)
	fmt.Println(string(text))
	conn, err := trusted.Dial(config.ProviderAddress, config.Credentials, config.ProviderToken)
	if err != nil {
		return nil, err
	}
	bytes, err := conn.Read()
	if err != nil {
		return nil, fmt.Errorf("could not get epoch from provider: %v", err)
	}
	epoch, _ := util.ParseUint64(bytes, 0)
	client := &DBClient{
		mu:    sync.Mutex{},
		Epoch: epoch,
		Conn:  conn,
		Queue: NewQueue(),
		Live:  true,
	}

	go func() {
		count := 0
		for {
			count += 1
			bytes, err := conn.Read()
			if err != nil {
				client.Live = false
				return
			}
			if len(bytes) == 8 { // && !config.KeepAlive {
				client.Finish, _ = util.ParseUint64(bytes, 0)
				client.Live = false
				client.Shutdown()
				conn.Shutdown()
				return
			}
			client.Queue.Add(bytes)
		}
	}()
	return client, nil
}

func (db *DBClient) Subscribe(epoch uint64, keepalive bool, tokens ...crypto.Token) error {
	msg := ReceiveTokens{
		Tokens:    tokens,
		FromEpoch: epoch,
	}
	return db.Conn.Send(msg.Serialize())
}
