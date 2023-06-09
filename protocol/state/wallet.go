package state

import (
	"encoding/binary"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/papirus"
)

func creditOrDebit(found bool, hash crypto.Hash, b *papirus.Bucket, item int64, param []byte) papirus.OperationResult {
	sign := int64(1)
	if param[0] == 1 {
		sign = -1 * sign
	}
	value := sign * int64(binary.LittleEndian.Uint64(param[1:]))
	if found {
		acc := b.ReadItem(item)
		balance := int64(binary.LittleEndian.Uint64(acc[crypto.Size:]))
		if value == 0 {
			return papirus.OperationResult{
				Result: papirus.QueryResult{Ok: true, Data: acc},
			}
		}
		newbalance := balance + value
		if newbalance > 0 {
			// update balance
			acc := make([]byte, crypto.Size+8)
			binary.LittleEndian.PutUint64(acc[crypto.Size:], uint64(newbalance))
			copy(acc[0:crypto.Size], hash[:])
			b.WriteItem(item, acc)
			return papirus.OperationResult{
				Result: papirus.QueryResult{Ok: true, Data: acc},
			}
		} else if newbalance == 0 {
			// account is market to be deleted
			return papirus.OperationResult{
				Deleted: &papirus.Item{Bucket: b, Item: item},
				Result:  papirus.QueryResult{Ok: true, Data: acc},
			}
		} else {
			return papirus.OperationResult{
				Result: papirus.QueryResult{Ok: false},
			}
		}
	} else {
		if value > 0 {
			acc := make([]byte, crypto.Size+8)
			binary.LittleEndian.PutUint64(acc[crypto.Size:], uint64(value))
			copy(acc[0:crypto.Size], hash[:])
			b.WriteItem(item, acc)
			return papirus.OperationResult{
				Added:  &papirus.Item{Bucket: b, Item: item},
				Result: papirus.QueryResult{Ok: false, Data: acc},
			}
		} else {
			return papirus.OperationResult{
				Result: papirus.QueryResult{
					Ok: false,
				},
			}
		}
	}
}

type Wallet struct {
	hs *papirus.HashStore[crypto.Hash]
}

func (w *Wallet) CreditHash(hash crypto.Hash, value uint64) bool {
	response := make(chan papirus.QueryResult)
	param := make([]byte, 9)
	binary.LittleEndian.PutUint64(param[1:], value)
	ok, _ := w.hs.Query(papirus.Query[crypto.Hash]{Hash: hash, Param: param, Response: response})
	return ok
}

func (w *Wallet) Credit(token crypto.Token, value uint64) bool {
	hash := crypto.HashToken(token)
	return w.CreditHash(hash, value)
}

func (w *Wallet) BalanceHash(hash crypto.Hash) (bool, uint64) {
	response := make(chan papirus.QueryResult)
	param := make([]byte, 9)
	ok, data := w.hs.Query(papirus.Query[crypto.Hash]{Hash: hash, Param: param, Response: response})
	if ok {
		return true, binary.LittleEndian.Uint64(data[32:])
	}
	return false, 0
}

func (w *Wallet) Balance(token crypto.Token) (bool, uint64) {
	hash := crypto.HashToken(token)
	return w.BalanceHash(hash)
}

func (w *Wallet) DebitHash(hash crypto.Hash, value uint64) bool {
	response := make(chan papirus.QueryResult)
	param := make([]byte, 9)
	param[0] = 1
	binary.LittleEndian.PutUint64(param[1:], value)
	ok, _ := w.hs.Query(papirus.Query[crypto.Hash]{Hash: hash, Param: param, Response: response})
	return ok
}

func (w *Wallet) Debit(token crypto.Token, value uint64) bool {
	hash := crypto.HashToken(token)
	return w.DebitHash(hash, value)
}

func (w *Wallet) Close() bool {
	ok := make(chan bool)
	w.hs.Stop <- ok
	return <-ok
}

func NewMemoryWalletStore(epoch uint64, bitsForBucket int64) *Wallet {
	nbytes := 56 + int64(1<<bitsForBucket)*(40*6+8)
	bytestore := papirus.NewMemoryStore(nbytes)
	Bucketstore := papirus.NewBucketStore(40, 6, bytestore)
	w := &Wallet{
		hs: papirus.NewHashStore("wallet", Bucketstore, int(bitsForBucket), creditOrDebit),
	}
	w.hs.Start()
	return w
}

func NewFileWalletStore(filePath string, epoch uint64, bitsForBucket int64) *Wallet {
	nbytes := 56 + int64(1<<bitsForBucket)*(40*6+8)
	bytestore := papirus.NewFileStore(filePath, nbytes)
	Bucketstore := papirus.NewBucketStore(40, 6, bytestore)
	w := &Wallet{
		hs: papirus.NewHashStore("wallet", Bucketstore, int(bitsForBucket), creditOrDebit),
	}
	w.hs.Start()
	return w
}
