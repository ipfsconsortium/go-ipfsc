package eth

import (
	"bytes"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/crypto"
	eth "github.com/ipfsconsortium/gipc/eth"
	log "github.com/sirupsen/logrus"
)

const ensEthNameServiceAbi string = `
[{"constant":true,"inputs":[{"name":"node","type":"bytes32"}],"name":"resolver","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"node","type":"bytes32"}],"name":"owner","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"node","type":"bytes32"},{"name":"label","type":"bytes32"},{"name":"owner","type":"address"}],"name":"setSubnodeOwner","outputs":[],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"node","type":"bytes32"},{"name":"ttl","type":"uint64"}],"name":"setTTL","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"node","type":"bytes32"}],"name":"ttl","outputs":[{"name":"","type":"uint64"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"node","type":"bytes32"},{"name":"resolver","type":"address"}],"name":"setResolver","outputs":[],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"node","type":"bytes32"},{"name":"owner","type":"address"}],"name":"setOwner","outputs":[],"payable":false,"type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"name":"node","type":"bytes32"},{"indexed":false,"name":"owner","type":"address"}],"name":"Transfer","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"node","type":"bytes32"},{"indexed":true,"name":"label","type":"bytes32"},{"indexed":false,"name":"owner","type":"address"}],"name":"NewOwner","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"node","type":"bytes32"},{"indexed":false,"name":"resolver","type":"address"}],"name":"NewResolver","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"node","type":"bytes32"},{"indexed":false,"name":"ttl","type":"uint64"}],"name":"NewTTL","type":"event"}]
`
const ensResolverAbi string = `
[{"constant": true,"inputs": [{"name": "interfaceID","type": "bytes4"}],"name": "supportsInterface","outputs": [{"name": "","type": "bool"}],"payable": false,"type": "function"}, {"constant": false,"inputs": [{"name": "node","type": "bytes32"}, {"name": "key","type": "string"}, {"name": "value","type": "string"}],"name": "setText","outputs": [],"payable": false,"type": "function"}, {"constant": true,"inputs": [{"name": "node","type": "bytes32"}, {"name": "contentTypes","type": "uint256"}],"name": "ABI","outputs": [{"name": "contentType","type": "uint256"}, {"name": "data","type": "bytes"}],"payable": false,"type": "function"}, {"constant": false,"inputs": [{"name": "node","type": "bytes32"}, {"name": "x","type": "bytes32"}, {"name": "y","type": "bytes32"}],"name": "setPubkey","outputs": [],"payable": false,"type": "function"}, {"constant": true,"inputs": [{"name": "node","type": "bytes32"}],"name": "content","outputs": [{"name": "ret","type": "bytes32"}],"payable": false,"type": "function"}, {"constant": true,"inputs": [{"name": "node","type": "bytes32"}],"name": "addr","outputs": [{"name": "ret","type": "address"}],"payable": false,"type": "function"}, {"constant": true,"inputs": [{"name": "node","type": "bytes32"}, {"name": "key","type": "string"}],"name": "text","outputs": [{"name": "ret","type": "string"}],"payable": false,"type": "function"}, {"constant": false,"inputs": [{"name": "node","type": "bytes32"}, {"name": "contentType","type": "uint256"}, {"name": "data","type": "bytes"}],"name": "setABI","outputs": [],"payable": false,"type": "function"}, {"constant": true,"inputs": [{"name": "node","type": "bytes32"}],"name": "name","outputs": [{"name": "ret","type": "string"}],"payable": false,"type": "function"}, {"constant": false,"inputs": [{"name": "node","type": "bytes32"}, {"name": "name","type": "string"}],"name": "setName","outputs": [],"payable": false,"type": "function"}, {"constant": false,"inputs": [{"name": "node","type": "bytes32"}, {"name": "hash","type": "bytes32"}],"name": "setContent","outputs": [],"payable": false,"type": "function"}, {"constant": true,"inputs": [{"name": "node","type": "bytes32"}],"name": "pubkey","outputs": [{"name": "x","type": "bytes32"}, {"name": "y","type": "bytes32"}],"payable": false,"type": "function"}, {"constant": false,"inputs": [{"name": "node","type": "bytes32"}, {"name": "addr","type": "address"}],"name": "setAddr","outputs": [],"payable": false,"type": "function"}, {"inputs": [{"name": "ensAddr","type": "address"}],"payable": false,"type": "constructor"}, {"anonymous": false,"inputs": [{"indexed": true,"name": "node","type": "bytes32"}, {"indexed": false,"name": "a","type": "address"}],"name": "AddrChanged","type": "event"}, {"anonymous": false,"inputs": [{"indexed": true,"name": "node","type": "bytes32"}, {"indexed": false,"name": "hash","type": "bytes32"}],"name": "ContentChanged","type": "event"}, {"anonymous": false,"inputs": [{"indexed": true,"name": "node","type": "bytes32"}, {"indexed": false,"name": "name","type": "string"}],"name": "NameChanged","type": "event"}, {"anonymous": false,"inputs": [{"indexed": true,"name": "node","type": "bytes32"}, {"indexed": true,"name": "contentType","type": "uint256"}],"name": "ABIChanged","type": "event"}, {"anonymous": false,"inputs": [{"indexed": true,"name": "node","type": "bytes32"}, {"indexed": false,"name": "x","type": "bytes32"}, {"indexed": false,"name": "y","type": "bytes32"}],"name": "PubkeyChanged","type": "event"}, {"anonymous": false,"inputs": [{"indexed": true,"name": "node","type": "bytes32"}, {"indexed": true,"name": "indexedKey","type": "string"}, {"indexed": false,"name": "key","type": "string"}],"name": "TextChanged","type": "event"}]
`

/*
def namehash(name):
  if name == '':
    return '\0' * 32
  else:
    label, _, remainder = name.partition('.')
	return sha3(namehash(remainder) + sha3(label))
*/

func NameHash(name string) common.Hash {
	if name == "" {
		var zero common.Hash
		return zero
	}

	split := strings.SplitN(name, ".", 2)

	label := split[0]
	var remainder string
	if len(split) > 1 {
		remainder = split[1]
	}
	reminderHash := NameHash(remainder)

	return crypto.Keccak256Hash(
		reminderHash[:],
		crypto.Keccak256([]byte(label)),
	)
}

type ENSClient struct {
	root     *eth.Contract
	resolver abi.ABI
}

func New(client *eth.Web3Client, address *common.Address) (*ENSClient, error) {

	rootabi, err := abi.JSON(bytes.NewReader([]byte(ensEthNameServiceAbi)))
	root, err := eth.NewContract(client, &rootabi, nil, address)

	resolverabi, err := abi.JSON(bytes.NewReader([]byte(ensResolverAbi)))
	if err != nil {
		return nil, err
	}

	return &ENSClient{root, resolverabi}, err
}

func (e *ENSClient) Info(name string) (string, error) {

	info := ""

	namehash := NameHash(name)

	var resolverAddr common.Address
	var ownerAddr common.Address
	if err := e.root.Call(&resolverAddr, "resolver", namehash); err != nil {
		return "", err
	}
	if err := e.root.Call(&ownerAddr, "owner", namehash); err != nil {
		return "", err
	}

	info += "ENS-Owner: " + ownerAddr.Hex()
	info += "\nENS-Resolver: " + resolverAddr.Hex()

	return info, nil
}

func (e *ENSClient) Text(name, key string) (string, error) {

	namehash := NameHash(name)

	var addr common.Address
	if err := e.root.Call(&addr, "resolver", namehash); err != nil {
		return "", err
	}
	log.Debug("ENS ", name, " key is ", namehash.Hex(), " => resolver ", addr.Hex())
	resolver, err := eth.NewContract(e.root.Client(), &e.resolver, nil, &addr)
	if err != nil {
		return "", err
	}

	var text string
	if err := resolver.Call(&text, "text", namehash, key); err != nil {
		return "", err
	}

	log.Debug("ENS ", name, ":", name, " => ", text)
	return text, nil

}

func (e *ENSClient) SetText(name, key, text string) error {

	namehash := NameHash(name)

	var addr common.Address
	if err := e.root.Call(&addr, "resolver", namehash); err != nil {
		return err
	}
	log.Debug("ENS ", name, " key is ", namehash.Hex(), " => resolver ", addr.Hex())
	resolver, err := eth.NewContract(e.root.Client(), &e.resolver, nil, &addr)
	if err != nil {
		return err
	}

	_, _, err = resolver.SendTransactionSync(nil, 0, "setText", namehash, key, text)
	if err != nil {
		return err
	}

	return nil
}
