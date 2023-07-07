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



