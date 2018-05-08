# goic

An implementation of the IPFS consortium written in golang

**Pre-alpha stuff for now**

## Quick start

### Install 

- Install golang, see https://golang.org/
- Install goic with the following command `go get github.com/ipfsconsortium/gipc`
  - binary should be installed in `~/go/bin/gipc`

### Create config file ./gipc.yaml

```
keystore:
    account: <account to use, e.g: 0xda4224ea7910d9c56d2f947d63088a556437da41>
    path: <path to the keystore, eg: /Users/hello/Library/Ethereum/keystore>
    passwd: <password of the keystore, eg : 1111 >

contracts:
    IPFSProxy:
        JSONURL: json with abi and contract, see notes
        Address: the address where the contract is deployed

IPFS:
    APIURL: the URL of the IPFS api, eg: http://localhost:5001

Web3:
    RPCURL: the URL of the geth rpc, eg: ws://localhost:8546
    StartBlock : the starting block to process, e.g: 4090116

DB:
    Path: where do you want to have the local database, e.g. /tmp/goicdb
```

if you need to deploy the IPFSProxy, you need to set the proxy contract initial parameters in the `gipc.yaml`

```
contracts:
    IPFSProxy:
        JSONURL: json with abi and contract, see notes>
        Deploy:
            Members:
                - initial member 1, e.g. 0xda4224ea7910d9c56d2f947d63088a556437da41
                - ...
            Required: requiered members, eg 2
            PersistLimit: intial persist limit, eg 1
```

#### Notes
- to create a keystore you can use `geth account new`
- for the IPFSProxy JSONURL, use https://raw.githubusercontent.com/ipfsconsortium/IPFSConsortiumContracts/1b78f4e167aeeb71523b3bb80580c9b95107b696/build/contracts/IPFSProxy.json for now

### Initialize

- run `gipc initdb`

### Deploy (install) the proxy smartcontract

- run `gipc proxydeploy`
- Set the address to the `contracts.IPFSProxy.Address` variable in `gipc.yaml`

### Start the server

- run `gipc serve`

## Operation

```
        _
  __ _ (_) _ __    ___
 / _` || || '_ \  / __|
| (_| || || |_) || (__
 \__, ||_|| .__/  \___|
 |___/    |_|
IPFS Consortium go implementation.

Usage:
  gipc [flags]
  gipc [command]

Available Commands:
  addhash         Add hash (send transaction)
  deployproxy     Deploy the proxy smartcontract (send transaction)
  dumpdb          Dumps the database
  help            Help about any command
  initdb          Initializes the database
  rmhash          Remove a hash (send transaction)
  serve           Start the server
  setpersistlimit Sets the persist limit (send transaction)
  skiptx          Skip a transaction

Flags:
      --config string    config file
  -h, --help             help for gipc
      --verbose string   verbose level (default "INFO")

Use "gipc [command] --help" for more information about a command.

```

- `gipc --verbose=DEBUG` to show more info
- `gipc dumpdb` *the server must be stopped*
- `gipc skiptx <txhash>` skips the processing of a transaction. If for whatever reason a transacton event cannot be processed, the server stops, this is by design. You must verify that all is ok, and specify that this transaction should be avoided with this command. *the server must be stopped*






