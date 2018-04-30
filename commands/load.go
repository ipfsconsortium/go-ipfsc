package commands

import (
	"fmt"
	"net/http"
	"sync"

	cfg "github.com/ipfsconsortium/gipc/config"
	eth "github.com/ipfsconsortium/gipc/eth"
	sto "github.com/ipfsconsortium/gipc/storage"
	log "github.com/sirupsen/logrus"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"

	shell "github.com/ipfs/go-ipfs-api"
)

var (
	client   *eth.Web3Client
	contract *eth.Contract
	ipfs     *shell.Shell
	storage  *sto.Storage
)

func load(withStorage bool) error {

	if client != nil {
		// already initialized
		return nil
	}

	var err error

	if err = loadEthClient(); err != nil {
		return err
	}
	if err = loadIPFSClient(); err != nil {
		return err
	}
	if err = loadContract(); err != nil {
		return err
	}
	if withStorage {
		if err = loadStorage(); err != nil {
			return err
		}
	}
	return nil
}

func loadStorage() error {

	var err error

	storage, err = sto.New(cfg.C.DB.Path)

	return err

}

func loadEthClient() error {

	var err error

	ks := keystore.NewKeyStore(cfg.C.Keystore.Path, keystore.StandardScryptN, keystore.StandardScryptP)
	account, err := ks.Find(accounts.Account{
		Address: common.HexToAddress(cfg.C.Keystore.Account),
	})
	if err != nil {
		return err
	}

	err = ks.Unlock(account, cfg.C.Keystore.Passwd)
	if err != nil {
		return err
	}

	client, err = eth.NewWeb3Client(
		cfg.C.Web3.RPCURL,
		ks,
		account,
	)
	if err != nil {
		return err
	}

	client.ClientMutex = &sync.Mutex{}

	balance, err := client.BalanceInfo()
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"account": account.Address.Hex(),
		"balance": balance,
	}).Info("Account loaded from keystore")
	return nil
}

func loadContract() error {

	var err error

	resp, err := http.Get(cfg.C.Contracts.IPFSProxy.JSONURL)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if cfg.C.Contracts.IPFSProxy.Address != "" {
		address := common.HexToAddress(cfg.C.Contracts.IPFSProxy.Address)
		contract, err = eth.NewContractFromJson(client, resp.Body, &address)
	} else {
		contract, err = eth.NewContractFromJson(client, resp.Body, nil)
	}
	return err

}

func loadIPFSClient() error {
	ipfs = shell.NewShell(cfg.C.IPFS.APIURL)
	if !ipfs.IsUp() {
		return fmt.Errorf("Cannot connect with local IPFS node")
	}
	return nil
}
