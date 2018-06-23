package eth

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"math/big"

	abi "github.com/ethereum/go-ethereum/accounts/abi"
	common "github.com/ethereum/go-ethereum/common"
	types "github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
)

// Contract is a smartcontract with optional address
type Contract struct {
	abi      *abi.ABI
	client   *Web3Client
	byteCode []byte
	address  *common.Address
}

var (
	errAddressHasNoCode = errors.New("address has no code")
)

func UnmarshallSolcAbiJson(jsonReader io.Reader) (*abi.ABI, []byte, error) {

	content, err := ioutil.ReadAll(jsonReader)
	if err != nil {
		return nil, nil, err
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(content, &fields); err != nil {
		return nil, nil, err
	}

	abivalue, bytecodehex := fields["abi"], fields["bytecode"].(string)

	byteCode, err := hex.DecodeString(bytecodehex[2:])
	if err != nil {
		return nil, nil, err
	}

	abijson, err := json.Marshal(&abivalue)
	if err != nil {
		return nil, nil, err
	}

	abiObject, err := abi.JSON(bytes.NewReader(abijson))
	if err != nil {
		return nil, nil, err
	}

	return &abiObject, byteCode, nil
}

// NewContract initiates a contract ABI & bytecode from json file associated to a web3 client
func NewContract(client *Web3Client, abi *abi.ABI, byteCode []byte, address *common.Address) (*Contract, error) {

	return &Contract{
		client:   client,
		abi:      abi,
		byteCode: byteCode,
		address:  address,
	}, nil
}

// NewContractFromJson initiates a contract ABI & bytecode from json file associated to a web3 client
func NewContractFromJson(client *Web3Client, solcjson io.Reader, address *common.Address) (*Contract, error) {

	abi, byteCode, err := UnmarshallSolcAbiJson(solcjson)
	if err != nil {
		return nil, err
	}

	return NewContract(client, abi, byteCode, address)
}

// VerifyBytecode verifies is the bytecode is the same than the JSON
func (c *Contract) VerifyBytecode() error {

	code, err := c.client.Client.CodeAt(context.TODO(), *c.address, nil)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"address":  c.address.Hex(),
		"codesize": len(code),
	}).Debug("CONTRACT get code size")

	if code == nil || len(code) == 0 {
		return errAddressHasNoCode
	}
	return nil
}

// SendTransactionSync executes a contract method and wait it finalizes
func (c *Contract) SendTransactionSync(value *big.Int, gasLimit uint64, funcname string, params ...interface{}) (*types.Transaction, *types.Receipt, error) {

	msg, err := c.abi.Pack(funcname, params...)
	if err != nil {
		log.Println("Failed packing ", funcname)
		return nil, nil, err
	}
	tx, receipt, err := c.client.SendTransactionSync(c.address, value, gasLimit, msg)
	if err != nil {
		log.Println("Failed calling ", funcname)
	}

	return tx, receipt, err
}

// Deploy the contract
func (c *Contract) DeploySync(params ...interface{}) (*types.Transaction, *types.Receipt, error) {

	init, err := c.abi.Pack("", params...)
	if err != nil {
		return nil, nil, err
	}

	code := append([]byte(nil), c.byteCode...)
	code = append(code, init...)

	tx, receipt, err := c.client.SendTransactionSync(nil, big.NewInt(0), 0, code)

	if err == nil {
		c.address = &receipt.ContractAddress
	}

	return tx, receipt, err
}

// Call an constant method
func (c *Contract) Call(ret interface{}, funcname string, params ...interface{}) error {

	input, err := c.abi.Pack(funcname, params...)
	if err != nil {
		return err
	}
	output, err := c.client.Call(c.address, big.NewInt(0), input)
	if err != nil {
		return err
	}
	return c.abi.Unpack(ret, funcname, output)
}

func (c *Contract) Abi() *abi.ABI {
	return c.abi
}

func (c *Contract) Client() *Web3Client {
	return c.client
}

func (c *Contract) ByteCode() []byte {
	return c.byteCode
}

func (c *Contract) Address() *common.Address {
	return c.address
}
