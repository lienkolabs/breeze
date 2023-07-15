# Beat

Beat is the default validating node for the breeze network. It deploys the 
swell consensus protocol (comming soon) or a proof-of-authority engine. 


## Building the source

Building beat requires both a Go (version 1.19 or later) compiler. 

With go compiler installed just run the command in the beat directory

```shell
$ go build 
```

## Running beat

### Configuration (proof-of-authority)

In order to run beat you need to set up a configuration json file with the 
following details:

```json
{
    "gatewayPort": number,
    "blockBroadcastPort": number,
    "walletDataPath": string,
    "secureVaultPath": string,
    "nodeToken": hex-string
}
```

The proof-of-authority node uses two ports. The node listens to both to
establish new connectiosn. Connections are signed but naked. These two ports 
must be accessible from the internet to make the node open. 

The gateway port is reserved for connections that sends actions to be validated
by the breeze network. 

The block broadcast port is reserver for connections that wishes to receive
a stream o newly minted blocks. Within breeze architecture the validating node
does not have the entire history of the block chain, but only recent ones that
might be necessary for disaster recovery scenarios.

### Managing Node

(comming soon)