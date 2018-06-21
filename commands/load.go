package commands

import (
	"context"
	"fmt"

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
	ipfs       *shell.Shell
	storage    *sto.Storage
	ens        *eth.ENSClient
)

func load(withStorage bool) error {

	if ipfs != nil {
		// already initialized
		return nil
	}

	var err error

	if withStorage {
		if err = loadStorage(); err != nil {
			return err
		}
	}

	if err = loadEthClients(); err != nil {
		return err
	}
	if err = loadIPFSClient(); err != nil {
		return err
	}
	if err = loadENS(); err != nil {
		return err
	}

	return nil
}

func loadStorage() error {

	var err error

	storage, err = sto.New(cfg.C.DB.Path)

	return err

}

func loadENS() error {

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

	ensClient := ethclients[cfg.C.EnsNames.Network]
	ensAddr := common.HexToAddress(cfg.C.Networks[cfg.C.EnsNames.Network].EnsRoot)

	web3 := eth.NewWeb3Client(ensClient, ks, &account)
	ens, err = eth.NewENSClient(web3, &ensAddr)
	return err
}
func loadEthClients() error {

	// load all clients

	ethclients = make(map[uint64]*ethclient.Client)

	for networkid, network := range cfg.C.Networks {

		log.WithField("url", network.RPCURL).Info("Checking WEB3.")

		client, err := ethclient.Dial(network.RPCURL)
		if err != nil {
			return err
		}

		clientnetworkid, err := client.NetworkID(context.Background())
		if err != nil {
			return err
		}

		if clientnetworkid.Uint64() != networkid {
			return fmt.Errorf("NetworkID RPC return a different networkid", networkid)
		}

		ethclients[networkid] = client

	}

	return nil
}

func loadIPFSClient() error {
	log.WithField("url", cfg.C.IPFS.APIURL).Info("Checking IPFS.")

	ipfs = shell.NewShell(cfg.C.IPFS.APIURL)
	if !ipfs.IsUp() {
		return fmt.Errorf("Cannot connect with local IPFS node")
	}
	return nil
}
