package protocol

/*
type Block struct {
	epoch          uint64
	Parent         crypto.Hash
	CheckPoint     uint64
	Publisher      crypto.Token
	PublishedAt    time.Time
	Actions        [][]byte
	Hash           crypto.Hash
	FeesCollected  uint64
	Signature      crypto.Signature
	validator      *state.MutatingState
	blockMutations *state.Mutations
}

func NewBlock(parent crypto.Hash, checkpoint, epoch uint64, publisher crypto.Token, validator *state.MutatingState) *Block {
	return &Block{
		Parent:         parent,
		epoch:          epoch,
		CheckPoint:     checkpoint,
		Publisher:      publisher,
		Actions:        make([][]byte, 0),
		validator:      validator,
		blockMutations: state.NewMutations(),
	}
}

func (b *Block) Incorporate(action actions.Action) bool {
	payments := action.Payments()
	if !b.CanPay(payments) {
		return false
	}
	b.TransferPayments(payments)
	b.Actions = append(b.Actions, action.Serialize())
	return true
}

func (b *Block) CanPay(payments *actions.Payment) bool {
	for _, debit := range payments.Debit {
		existingBalance := b.validator.Balance(debit.Account)
		delta := b.blockMutations.DeltaBalance(debit.Account)
		if int(existingBalance) < int(debit.FungibleTokens)+delta {
			return false
		}
	}
	return true
}

func (b *Block) CanWithdraw(hash crypto.Hash, value uint64) bool {
	existingBalance := b.validator.Balance(hash)
	return value < existingBalance
}

func (b *Block) Deposit(hash crypto.Hash, value uint64) {
	if old, ok := b.validator.Mutations.DeltaDeposits[hash]; ok {
		b.validator.Mutations.DeltaDeposits[hash] = old + int(value)
		return
	}
	b.validator.Mutations.DeltaDeposits[hash] = int(value)
}

func (b *Block) TransferPayments(payments *actions.Payment) {
	for _, debit := range payments.Debit {
		if delta, ok := b.blockMutations.DeltaWallets[debit.Account]; ok {
			b.blockMutations.DeltaWallets[debit.Account] = delta - int(debit.FungibleTokens)
		} else {
			b.blockMutations.DeltaWallets[debit.Account] = -int(debit.FungibleTokens)
			// fmt.Println(debit.Account, debit.FungibleTokens)
		}
	}
	for _, credit := range payments.Credit {
		if delta, ok := b.blockMutations.DeltaWallets[credit.Account]; ok {
			b.blockMutations.DeltaWallets[credit.Account] = delta + int(credit.FungibleTokens)
		} else {
			b.blockMutations.DeltaWallets[credit.Account] = int(credit.FungibleTokens)
		}
	}
}

func (b *Block) Balance(hash crypto.Hash) uint64 {
	return b.validator.Balance(hash)
}

func (b *Block) AddFeeCollected(value uint64) {
	b.FeesCollected += value
}

func (b *Block) Epoch() uint64 {
	return b.epoch
}

func (b *Block) Seal(token crypto.PrivateKey) {
	b.PublishedAt = time.Now()
	bytes := b.serializeForHash()
	b.Hash = crypto.Hasher(bytes)
	b.Sign(token)
}

func (b *Block) Sign(token crypto.PrivateKey) {
	b.Signature = token.Sign(b.serializeWithoutSignature())
}

func (b *Block) Serialize() []byte {
	bytes := b.serializeWithoutSignature()
	util.PutSignature(b.Signature, &bytes)
	util.PutByteArray(b.Hash[:], &bytes)
	return bytes
}

func (b *Block) serializeForHash() []byte {
	bytes := make([]byte, 0)
	util.PutUint64(b.epoch, &bytes)
	util.PutByteArray(b.Parent[:], &bytes)
	util.PutUint64(b.CheckPoint, &bytes)
	util.PutByteArray(b.Publisher[:], &bytes)
	util.PutTime(b.PublishedAt, &bytes)
	util.PutUint16(uint16(len(b.Actions)), &bytes)
	for _, action := range b.Actions {
		util.PutByteArray(action, &bytes)
	}
	util.PutUint64(b.FeesCollected, &bytes)
	return bytes
}

func (b *Block) serializeWithoutSignature() []byte {
	bytes := b.serializeForHash()
	util.PutByteArray(b.Hash[:], &bytes)
	return bytes
}

func ParseBlock(data []byte) *Block {
	position := 0
	block := Block{}
	block.epoch, position = util.ParseUint64(data, position)
	block.Parent, position = util.ParseHash(data, position)
	block.CheckPoint, position = util.ParseUint64(data, position)
	block.Publisher, position = util.ParseToken(data, position)
	block.PublishedAt, position = util.ParseTime(data, position)
	block.Actions, position = util.ParseByteArrayArray(data, position)
	block.Hash, position = util.ParseHash(data, position)
	block.FeesCollected, position = util.ParseUint64(data, position)
	msg := data[0:position]
	block.Signature, _ = util.ParseSignature(data, position)
	if !block.Publisher.Verify(msg, block.Signature) {
		fmt.Println("wrong signature")
		return nil
	}
	block.blockMutations = state.NewMutations()
	return &block
}

func (b *Block) SetValidator(validator *state.MutatingState) {
	b.validator = validator
}

func GetBlockEpoch(data []byte) uint64 {
	if len(data) < 8 {
		return 0
	}
	epoch, _ := util.ParseUint64(data, 0)
	return epoch
}

func (b *Block) JSONSimple() string {
	bulk := &util.JSONBuilder{}
	bulk.PutUint64("epoch", b.epoch)
	bulk.PutHex("parent", b.Parent[:])
	bulk.PutUint64("checkpoint", b.CheckPoint)
	bulk.PutHex("publisher", b.Publisher[:])
	bulk.PutTime("publishedAt", b.PublishedAt)
	bulk.PutUint64("actionsCount", uint64(len(b.Actions)))
	bulk.PutHex("hash", b.Parent[:])
	bulk.PutUint64("feesCollectes", b.FeesCollected)
	bulk.PutBase64("signature", b.Signature[:])
	return bulk.ToString()
}
*/
