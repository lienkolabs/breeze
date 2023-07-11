package store

import (
	"errors"
	"fmt"
	"os"

	"github.com/lienkolabs/breeze/crypto"
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
	fileNameTemplate string
	files            []*os.File
	length           int
	current          int
	index            map[crypto.Token][]DBMessage
	tokeninzer       Tokenizer
}

func NewDB(broker chan []byte, fileNameTemplate string, tokenizer Tokenizer) (*DB, error) {
	if tokenizer == nil {
		return nil, errors.New("a valid tokenizer function must be provided")
	}
	if fmt.Sprintf(fileNameTemplate, 1) == fmt.Sprintf(fileNameTemplate, 1) {
		return nil, errors.New("file template must contain a %v wildcard")
	}
	db := &DB{
		fileNameTemplate: fileNameTemplate,
		files:            make([]*os.File, 0),
		index:            make(map[crypto.Token][]DBMessage),
		tokeninzer:       tokenizer,
	}
	db.CreateNewFile()
	return db, nil
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
