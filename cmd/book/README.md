# BOOK

Book is a simple indexation engine and blockchain data store for the breeze
network. It keeps tracks of all blocks produced by the breeze network consensus
and serve this information to interested parties.

## Building book

Building beat requires both a Go (version 1.19 or later) compiler. 

With go compiler installed just run the command in the beat directory

```shell
$ go build 
```

## Running book

### Configuration

```json
{
	"ServePort": number,
	"BlockServiceAddress": url-to-block-provider,
	"BlockServiveToken": hex-string-of-block-provider-token,
	"FileNameTemplate": "any_name%v.any_ext",
    "NodeToken": hex-string-of-node,
    "SecureVaultPath": path-to-secure-vault
}
```
