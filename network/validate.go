package network

import "github.com/lienkolabs/breeze/crypto"

type ValidateConnection interface {
	ValidateConnection(token crypto.Token) chan bool
}

type acceptAll struct{}

func (a acceptAll) ValidateConnection(token crypto.Token) chan bool {
	response := make(chan bool)
	go func() {
		response <- true
	}()
	return response
}

// An implementation with ValidateConnection interface that accepts all reequested
// connections.
var AcceptAllConnections = acceptAll{}
