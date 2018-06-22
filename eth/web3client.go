package eth

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	log "github.com/sirupsen/logrus"
)

var (
	// ErrReceiptStatusFailed when recieving a failed transaction
	errReceiptStatusFailed = errors.New("receipt status is failed")
	// ErrReceiptNotRecieved when unable to retrieve a transaction
	errReceiptNotRecieved = errors.New("receipt not available")
)

// Web3Client defines a connection to a client via websockets
type Web3Client struct {
	ClientMutex    *sync.Mutex
	Client         *ethclient.Client
	Account        *accounts.Account
	Ks             *keystore.KeyStore
	ReceiptTimeout time.Duration
	MaxGasPrice    uint64
}

// NewWeb3Client creates a client, using a keystore and an account for transactions
func NewWeb3ClientWithURL(rpcURL string, ks *keystore.KeyStore, account *accounts.Account) (*Web3Client, error) {

	var err error

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}

	return &Web3Client{
		Client:         client,
		Ks:             ks,
		Account:        account,
		ReceiptTimeout: 120 * time.Second,
	}, nil
}

// NewWeb3Client creates a client, using a keystore and an account for transactions
func NewWeb3Client(client *ethclient.Client, ks *keystore.KeyStore, account *accounts.Account) *Web3Client {

	return &Web3Client{
		Client:         client,
		Ks:             ks,
		Account:        account,
		ReceiptTimeout: 120 * time.Second,
		MaxGasPrice:    4000000000,
	}
}

// BalanceInfo retieves information about the default account
func (w *Web3Client) BalanceInfo() (string, error) {

	ctx := context.TODO()
	balance, err := w.Client.BalanceAt(ctx, w.Account.Address, nil)
	if err != nil {

		return "", err
	}
	return balance.String(), nil
}

// SendTransactionSync executes a contract method and wait it finalizes
func (w *Web3Client) SendTransactionSync(to *common.Address, value *big.Int, gasLimit uint64, calldata []byte) (*types.Transaction, *types.Receipt, error) {

	w.ClientMutex.Lock()
	defer w.ClientMutex.Unlock()

	var err error
	var tx *types.Transaction
	var receipt *types.Receipt

	ctx := context.TODO()

	if value == nil {
		value = big.NewInt(0)
	}

	network, err := w.Client.NetworkID(ctx)
	if err != nil {
		return nil, nil, err
	}

	gasPrice, err := w.Client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, nil, err
	}

	if gasPrice.Uint64() > w.MaxGasPrice {
		log.Error("WEB3 Failed EstimateGas from=%v to=%v value=%v data=%v")
		return nil, nil, fmt.Errorf("Max gas price reached %v > %v", gasPrice, w.MaxGasPrice)
	}

	callmsg := ethereum.CallMsg{
		From:  w.Account.Address,
		To:    to,
		Value: value,
		Data:  calldata,
	}

	if gasLimit == 0 {
		gasLimit, err = w.Client.EstimateGas(ctx, callmsg)
		if err != nil {
			log.Error("WEB3 Failed EstimateGas from=%v to=%v value=%v data=%v",
				callmsg.From.Hex(), callmsg.To.Hex(),
				callmsg.Value, hex.EncodeToString(callmsg.Data),
			)
			return nil, nil, err
		}
	}

	nonce, err := w.Client.NonceAt(ctx, w.Account.Address, nil)
	if err != nil {
		return nil, nil, err
	}

	if to == nil {
		tx = types.NewContractCreation(
			nonce,    // nonce int64
			value,    // amount *big.Int
			gasLimit, // gasLimit *big.Int
			gasPrice, // gasPrice *big.Int
			calldata, // data []byte
		)
	} else {
		tx = types.NewTransaction(
			nonce,    // nonce int64
			*to,      // to common.Address
			value,    // amount *big.Int
			gasLimit, // gasLimit *big.Int
			gasPrice, // gasPrice *big.Int
			calldata, // data []byte
		)
	}

	if tx, err = w.Ks.SignTx(*w.Account, tx, network); err != nil {
		return nil, nil, err
	}

	log.WithFields(log.Fields{
		"tx":       tx.Hash().Hex(),
		"gasprice": fmt.Sprintf("%.2f Gwei", float64(tx.GasPrice().Uint64())/1000000000.0),
	}).Info("WEB3 Sending transaction")
	if err = w.Client.SendTransaction(ctx, tx); err != nil {
		return nil, nil, err
	}

	start := time.Now()
	for receipt == nil && time.Now().Sub(start) < w.ReceiptTimeout {
		receipt, err = w.Client.TransactionReceipt(ctx, tx.Hash())
		if receipt == nil {
			time.Sleep(200 * time.Millisecond)
		}
	}

	if receipt != nil && receipt.Status == types.ReceiptStatusFailed {
		log.WithField("tx", tx.Hash().Hex()).Error("WEB3 Failed transaction receipt")
		return tx, receipt, errReceiptStatusFailed
	}

	if receipt == nil {
		log.WithField("tx", tx.Hash().Hex()).Error("WEB3 Failed transaction")
		return tx, receipt, errReceiptNotRecieved
	}
	log.WithField("tx", tx.Hash().Hex()).Debug("WEB3 Success transaction")

	return tx, receipt, err
}

// Call an constant method
func (w *Web3Client) Call(to *common.Address, value *big.Int, calldata []byte) ([]byte, error) {

	ctx := context.TODO()

	msg := ethereum.CallMsg{
		From:  w.Account.Address,
		To:    to,
		Value: value,
		Data:  calldata,
	}

	return w.Client.CallContract(ctx, msg, nil)
}

// Do a web3 signature
func (w *Web3Client) Sign(data ...[]byte) ([3][32]byte, error) {
	web3SignaturePrefix := []byte("\x19Ethereum Signed Message:\n32")

	hash := crypto.Keccak256(data...)
	prefixedHash := crypto.Keccak256(web3SignaturePrefix, hash)

	var ret [3][32]byte

	// The produced signature is in the [R || S || V] format where V is 0 or 1.
	sig, err := w.Ks.SignHash(*w.Account, prefixedHash)
	if err != nil {
		return ret, err
	}

	// We need to convert it to the format []uint256 = {v,r,s} format
	ret[0][31] = sig[64] + 27
	copy(ret[1][:], sig[0:32])
	copy(ret[2][:], sig[32:64])
	return ret, nil
}
