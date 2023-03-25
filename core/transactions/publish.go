package transactions

import (
	"github.com/lienkolabs/breeze/core/crypto"
	"github.com/lienkolabs/breeze/core/util"
)

type Publish struct {
	TimeStamp uint64
	Data      []byte
	Wallet    crypto.Token
	Fee       uint64
	Signature crypto.Signature
}

func (t *Publish) serializeSign() []byte {
	bytes := []byte{0, ITransfer}
	util.PutUint64(t.TimeStamp, &bytes)
	util.PutByteArray(t.Data, &bytes)
	util.PutToken(t.Wallet, &bytes)
	util.PutUint64(t.Fee, &bytes)
	return bytes
}

func (t *Publish) Serialize() []byte {
	bytes := t.serializeSign()
	util.PutSignature(t.Signature, &bytes)
	return bytes
}

func (t *Publish) Epoch() uint64 {
	return t.TimeStamp
}

func (t *Publish) Kind() byte {
	return IPublish
}

func (t *Publish) Debit() Wallet {
	return Wallet{Account: crypto.HashToken(t.Wallet), FungibleTokens: t.Fee}
}

func (t *Publish) Payments() *Payment {
	payment := &Payment{
		Credit: make([]Wallet, 0),
		Debit:  make([]Wallet, 0),
	}
	payment.NewDebit(crypto.HashToken(t.Wallet), t.Fee)
	return payment
}

func (t *Publish) Sign(key crypto.PrivateKey) {
	bytes := t.serializeSign()
	t.Signature = key.Sign(bytes)
}

func ParsePublish(data []byte) *Publish {
	if len(data) < 2 || data[1] != IPublish {
		return nil
	}
	p := Publish{}
	position := 2
	p.TimeStamp, position = util.ParseUint64(data, position)
	p.Data, position = util.ParseByteArray(data, position)
	p.Wallet, position = util.ParseToken(data, position)
	p.Fee, position = util.ParseUint64(data, position)
	msg := data[0:position]
	p.Signature, _ = util.ParseSignature(data, position)
	if !p.Wallet.Verify(msg, p.Signature) {
		return nil
	}
	return &p
}
