# goic

```
                     _          __
  __ _   ___        (_) _ __   / _| ___   ___
 / _` | / _ \  ____ | || '_ \ | |_ / __| / __|
| (_| || (_) ||____|| || |_) ||  _|\__ \| (__
 \__, | \___/       |_|| .__/ |_|  |___/ \___|
 |___/                 |_|
IPFS pinning consortium

Usage:
  gipc [flags]
  gipc [command]

Available Commands:
  add         Add hash to IPFS
  db-dump     Dumps the database
  db-init     Initializes the database
  help        Help about any command
  init        Initialize ipfsc
  ls          Info of local ens
  rm          Remove hash to IPFS
  sync-loop   Looping sync forever
  sync-once   Sync one shot

Flags:
      --config string    config file
  -h, --help             help for gipc
      --verbose string   verbose level (default "INFO")

Use "gipc [command] --help" for more information about a command.
```


**Pre-alpha stuff for now**

## Quick start

### Install 

- Install golang, see https://golang.org/
- Install goic with the following command `go get github.com/ipfsconsortium/go-ipfsc`
  - binary should be installed in `~/go/bin/go-ipfsc`

### Create config file ./gipc.yaml

```
keystore:
  account: <account to use, e.g: 0xda4224ea7910d9c56d2f947d63088a556437da41>
  path: <path to the keystore, eg: /Users/hello/Library/Ethereum/keystore>
  passwd: <password of the keystore, eg : 1111 >

ensnames:
  network: <network to use, 1 for maninnet>
  local: <your ENS domain>
  remotes:
    - <ENS domain containing IPFS manifest 1>
    - <ENS domain containing IPFS manifest 2>
    - ...

db:
  path: <where do you want to have the local database, e.g. /tmp/goicdb>

ipfs:
  apiurl: <the URL of the IPFS api, eg: http://localhost:5001>
  timeout: <time of IPFS connection timeout, e.g. 60s>

networks:
  <networkid, 1 for mainnet>: //
    maxgasprice: <max gas price to pay, e.g. 4000000000=4GWei>
    rpcurl : <URL of WEB3 HTTP API>  
    ensroot : <where ENS root is located, 0x314159265dd8dbb310642f98f50c066173c1259b for mainnet>

api:
  port: <port for the api web service, like 8991>
```

Note:  to create a keystore you can use `geth account new`

### Initialize the database

- `gipc db-init` 

### Set-up a new ENS IPFS Manifest entry

- `gipc init`

### Add/remove entries to the IPFS manifest

- `gipc add <ipfs hash>` or `gipc add <file path>`
- `gipc rm <ipfs hash>` 
 
### List IPFS entries in your ENS contract

- `gipc rm <ipfs hash>` 

### PIN other ENS IPFS manifest entries to your local IPFS

- `gipc sync-loop` (sync continuosuly) 
- `gipc sync-once` (sync one time) 

### Get the current stats

- go to `http://localhost:8991/stats`










