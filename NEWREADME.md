# BREEZE

Official implementation of the Breeze protocol.

## Building the source

## Executables

Breeze archtecture is a modular one. The project comes with several executables, each one implementing one specific functionality 
of the breeze protocol. These can be found in the `cmd` directory.

|  Command   | Description                                                                                                        |
| :--------: | ------------------------------------------------------------------------------------------------------------------ |
| **`beat`** | Implemenation of the validating node.                                                                              |
| `book`     | Block persitance. It offers historical information indexed by token.                                               |
| `link`     | Implementation of a gateway to the breeze validators network.                                                      |
| `safe`     | A simple wallet to safekeep secret credentials and gather information on their associated token from the network.  |

## Running `beat`

Usage:

```shell
$ beat path-to-config-file.json
```

### Configuration File

The structure of the configuration file is the following

```
{
    "gatewayPort": numeric,
    "blockBroadcastPort": numeric,
    "peer2peerPort": numeric,
    "stateDataFolder": string,
    "secureVaultFile": string,
    "consensusEngine": string,
    "genesis" : {
        "token": hex-token,
        "aero": numeric, 
    }
    "peerAddress": string
}
```



### Hardware requirements


## Social Protocols

The imagined use case for breeze network is the development of specialized social
protocol on top of it. 

### BLockchain composition

Breeze offer a generic action, called void, that has a 
single general purpose data field. The first four bytes of this field should be
reserved to the specification of the protocol that is being used. The remaining 
space is reserved for information relevant to that protocol.

### Consensus Logic

Any developer of a social protocol must understand at least the underlying logic
of block formation within breeze architecture. Blocks are formed in four steps:

* Liveblock. Block header and incoporated actions are broadcast to interested 
  parties by the breeze network. These actions are validated against a checkpoint
  that might not be the block immediately prior to the live one, since it might 
  not be available at the start of block formation.

* Sealed block. After the standard block interval (1 second) the block proposer 
  seals the block by timestamping it and providing a signature on it. No action
  can be included in a block after it was sealed, but some of those actions 
  included can be discarded.

* Consensus and commit. After the proposal of the block breeze network enters a 
  consensus formation phase, that might take a few ms or a few seconds depending
  on connectivity conditions or the presence of malicious nodes. Consensus might
  be achieved on a block other that the originally proposed one, in which case
  the network will be instructuted to roll back one or a few sealed blocks. 
  Rollback are only possible to non-commited blocks. Once a consensus
  is achieved the block is ready to be commited. On the commit phase, if the 
  checkpoint is not the prior block, the block is revalidated against the previous
  block, and if there were actions incorporated that are invalidated they are
  marked as such.

* Chekpoint. Every 15 minutes the system undergoes a chekpoint phase, where the 
  validator participation pool is redistributed and a checksum over the state 
  of the breeze network (the wallets balances and the consecutive hash of block 
  hashes). Checkpoint serves general purposes for the network, but of relevance 
  for social protocol developers is that it is a disaster recovery tool. If by 
  any means the network was corrupted by malicious actors, it might be rolled
  back to the checkpoint state notwithstanding commit blocks. 

### Deployment of Social Protocols

In practice the developer must provide a state and state mutations interface 
