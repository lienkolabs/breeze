package main

import (
	"fmt"
	"log"
	"time"

	"github.com/lienkolabs/breeze/crypto"
	"github.com/lienkolabs/breeze/network/echo"
	"github.com/lienkolabs/breeze/protocol/actions"
)

func Simulation(key crypto.PrivateKey, port int) {
	ticker := time.NewTicker(250 * time.Microsecond)
	for n := 0; n < 10; n++ {
		<-ticker.C
	}
	host := fmt.Sprintf("localhost:%v", port)
	server, err := echo.NewActionServer(host, key, key.PublicKey())
	if err != nil {
		log.Fatalf("could not connect to gateway: %v", err)
	}
	count := uint64(0)
	for {
		<-ticker.C
		count++
		address, _ := crypto.RandomAsymetricKey()
		transfer := actions.Transfer{
			TimeStamp: count / 10,
			From:      key.PublicKey(),
			To: []crypto.TokenValue{
				{
					Token: address,
					Value: 1,
				},
			},
			Reason: "Testing",
			Fee:    0,
		}
		transfer.Sign(key)
		server.Send(transfer.Serialize())
	}
}
