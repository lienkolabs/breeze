package store

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network/echo"
	"github.com/lienkolabs/breeze/protocol/chain"
	"github.com/lienkolabs/breeze/util"
)

const maxFileLength = 1 << 32

//const fileNameTemaplate = "breeze_rawdb_%v.dat"

type Tokenizer func([]byte) []crypto.Token

type DBMessage struct {
	file     int
	block    uint64
	position int
	size     int
}

type DB struct {
	epoch            uint64
	mu               sync.Mutex
	fileNameTemplate string
	files            []*os.File
	length           int
	current          int
	index            map[crypto.Token][]DBMessage
	tokeninzer       Tokenizer
	jobs             map[crypto.Token]*echo.NewIndexJob
	runningJob       map[*echo.NewIndexJob]struct{}
}

func (db *DB) Close() {
	for _, file := range db.files {
		file.Close()
	}
}

func NewDB(fileNameTemplate string, tokenizer Tokenizer) (*DB, error) {
	if tokenizer == nil {
		return nil, errors.New("a valid tokenizer function must be provided")
	}
	if fmt.Sprintf(fileNameTemplate, 1) == fmt.Sprintf(fileNameTemplate, 2) {
		return nil, errors.New("file template must contain a %v wildcard")
	}
	db := &DB{
		mu:               sync.Mutex{},
		fileNameTemplate: fileNameTemplate,
		files:            make([]*os.File, 0),
		index:            make(map[crypto.Token][]DBMessage),
		tokeninzer:       tokenizer,
		jobs:             make(map[crypto.Token]*echo.NewIndexJob),
		runningJob:       make(map[*echo.NewIndexJob]struct{}),
	}
	db.CreateNewFile()
	return db, nil
}

func (db *DB) AppendJob(job *echo.NewIndexJob) {
	if job.KeepAlive {
		db.mu.Lock()
		for _, token := range job.Tokens {
			db.jobs[token] = job
		}
		db.mu.Unlock()
	}
	db.StartJob(job, db.epoch)
}

func (db *DB) StartJob(job *echo.NewIndexJob, endEpoch uint64) {
	go func() {
		messagepool := make(map[DBMessage]struct{})
		for _, token := range job.Tokens {
			for _, dbMessage := range db.index[token] {
				if dbMessage.block >= job.FromEpoch && dbMessage.block <= endEpoch {
					messagepool[dbMessage] = struct{}{}
				}
			}
		}
		for msg := range messagepool {
			data := db.ReadMessage(msg)
			job.Connection.Send(data)
		}
		delete(db.runningJob, job)
		if !job.KeepAlive {
			bytes := make([]byte, 8)
			util.PutUint64(endEpoch, &bytes)
			job.Connection.Send(bytes)
		}
	}()
}

func (db *DB) AppendBlock(block *chain.Block) (*DBMessage, error) {
	blockBytes := block.Serialize()
	sizeBytes := make([]byte, 0)
	util.PutUint32(uint32(len(blockBytes)), &sizeBytes)
	blockBytes = append(sizeBytes, blockBytes...)
	dataLen, err := db.files[db.current-1].Write(blockBytes)
	if err != nil || dataLen < len(blockBytes) {
		return nil, fmt.Errorf("could not persist message on the database: %v", err)
	}
	position := db.length // position at the start of the block
	fileNum := len(db.files) + 1
	db.length += dataLen
	if db.length > maxFileLength {
		db.CreateNewFile()
	}
	return &DBMessage{file: fileNum, position: position, size: len(blockBytes)}, nil
}

func (db *DB) IncorporateBlock(block *chain.Block) error {
	blockMessage, err := db.AppendBlock(block)
	if err != nil {
		return err
	}
	db.epoch = block.Epoch
	head := block.Header()
	position := blockMessage.position + len(head) + 4
	for _, action := range block.Actions {
		dbMessage := DBMessage{
			file:     db.current,
			block:    block.Epoch,
			position: position + 2, // 2 for the size of the action
			size:     len(action),
		}
		tokens := db.tokeninzer(action)
		if len(tokens) > 0 {
			for _, token := range tokens {
				if index, ok := db.index[token]; ok {
					db.index[token] = append(index, dbMessage)
				} else {
					db.index[token] = []DBMessage{dbMessage}
				}
			}
		}
		position += len(action) + 2
	}
	return nil
}

func (db *DB) ReadMessage(msg DBMessage) []byte {
	file := db.files[msg.file]
	data := make([]byte, msg.size)
	bytes, err := file.ReadAt(data, int64(msg.position))
	if err != nil || bytes != msg.size {
		return nil
	}
	return data
}

func (db *DB) CreateNewFile() {
	if len(db.files) > 0 {
		current := db.files[len(db.files)-1]
		if err := current.Close(); err != nil {
			panic(err)
		}
		// reopen as readonly
		filepath := current.Name()
		if file, err := os.Open(filepath); err != nil {
			panic(err)
		} else {
			db.files[len(db.files)-1] = file
		}
	}
	if newFile, err := os.Create(fmt.Sprintf(db.fileNameTemplate, len(db.files)+1)); err != nil {
		panic(err)
	} else {
		db.files = append(db.files, newFile)
		db.length = 0
		db.current += 1
	}
}
