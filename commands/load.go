package commands

import (
	"context"
	"fmt"
	"sync"

	cfg "github.com/ipfsconsortium/gipc/config"
	ens "github.com/ipfsconsortium/gipc/ens"
	eth "github.com/ipfsconsortium/gipc/eth"
	ipfsclient "github.com/ipfsconsortium/gipc/ipfsc"
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
	ipfsc      *ipfsclient.Ipfsc
	storage    *sto.Storage
)

func load(withStorage bool) error {

	if ipfsc != nil {
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

	return loadIPFSC()

}

func loadStorage() error {

	var err error

	storage, err = sto.New(cfg.C.DB.Path)

	return err

}

func loadIPFSC() error {

	// load ens.

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
	web3.ClientMutex = &sync.Mutex{}
	ensclient, err := ens.New(web3, &ensAddr)
	if err != nil {
		return err
	}

	log.WithField("url", cfg.C.IPFS.APIURL).Info("Checking IPFS.")

	// load ipfs

	ipfs := shell.NewShell(cfg.C.IPFS.APIURL)
	if !ipfs.IsUp() {
		return fmt.Errorf("Cannot connect with local IPFS node")
	}

	ipfsc = ipfsclient.New(ipfs, ensclient)

	return nil

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
