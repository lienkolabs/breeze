package consensus

import (
	"github.com/lienkolabs/breeze/core/block"
	"github.com/lienkolabs/breeze/core/transactions"
)

func ValidateBlock(data []byte, validator block.MutatingState) *block.Block {
	validationBlock := block.ParseBlock(data)
	validationBlock.SetValidator(&validator)
	for _, instructionBytes := range validationBlock.Instructions {
		instruction := transactions.ParseTransaction(instructionBytes)
		if instruction == nil {
			return nil
		}
		if !validationBlock.Incorporate(instruction) {
			return nil
		}
	}
	return validationBlock
}
