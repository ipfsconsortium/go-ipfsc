package commands

import (
	"context"
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
	"github.com/ethereum/go-ethereum/ethclient"

	shell "github.com/adriamb/go-ipfs-api"
)

var (
	ethclients map[uint64]*ethclient.Client
	proxy      *eth.Contract
	ipfs       *shell.Shell
	storage    *sto.Storage
)

func load(withStorage bool) error {

	if proxy != nil {
		// already initialized
		return nil
	}

	var err error

	if err = loadEthClients(); err != nil {
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

func loadEthClients() error {

	// load all clients

	ethclients = make(map[uint64]*ethclient.Client)

	for _, network := range cfg.C.Networks {

		log.WithField("network", network.RPCURL).Info("Checking network.")

		client, err := ethclient.Dial(network.RPCURL)
		if err != nil {
			return err
		}

		networkid, err := client.NetworkID(context.Background())
		if err != nil {
			return err
		}

		if networkid.Uint64() != network.NetworkID {
			return fmt.Errorf("NetworkID RPC return a different networkid", network.NetworkID)
		}

		ethclients[network.NetworkID] = client

	}

	return nil
}

func loadContract() error {

	// load the keystore

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

	// crete web3 client & get info

	client, err := eth.NewWeb3Client(
		ethclients[cfg.C.Contracts.IPFSProxy.NetworkID],
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

	// create the contract

	resp, err := http.Get(cfg.C.Contracts.IPFSProxy.JSONURL)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if cfg.C.Contracts.IPFSProxy.Address != "" {
		address := common.HexToAddress(cfg.C.Contracts.IPFSProxy.Address)
		proxy, err = eth.NewContractFromJson(client, resp.Body, &address)
	} else {
		proxy, err = eth.NewContractFromJson(client, resp.Body, nil)
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
