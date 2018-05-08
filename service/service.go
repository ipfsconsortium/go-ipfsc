package service

import (
	"errors"
	"math/big"
	"os"
	"os/signal"

	"github.com/ethereum/go-ethereum/ethclient"

	shell "github.com/adriamb/go-ipfs-api"
	"github.com/ethereum/go-ethereum/common"
	eth "github.com/ipfsconsortium/gipc/eth"
	sto "github.com/ipfsconsortium/gipc/storage"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	persistLimit *big.Int
	proxyNetwork uint64
	ethclients   map[uint64]*ethclient.Client
	dispatchers  map[uint64]*ScanEventDispatcher
	proxy        *eth.Contract
	ipfs         *shell.Shell
	storage      *sto.Storage
}

var (
	errVerifySmartcontract = errors.New("cannot verify deployed smartcontract")
	errReadPersistLimit    = errors.New("error reading current persistLimit")
	errReachedPersistLimit = errors.New("persistlimit reached")
)

func NewService(ethclients map[uint64]*ethclient.Client, proxyNetwork uint64, proxy *eth.Contract, ipfs *shell.Shell, storage *sto.Storage) *Service {

	dispatchers := make(map[uint64]*ScanEventDispatcher)

	for networkid, ethclient := range ethclients {
		dispatchers[networkid] = NewScanEventDispatcher(ethclient, newSavePoint(storage, networkid))
	}

	return &Service{
		dispatchers:  dispatchers,
		proxyNetwork: proxyNetwork,
		ethclients:   ethclients,
		proxy:        proxy,
		ipfs:         ipfs,
		storage:      storage,
	}
}

func (s *Service) Serve() error {

	var err error

	// -- read current persistlimit

	if err = s.proxy.VerifyBytecode(); err != nil {
		return errVerifySmartcontract
	}

	if err = s.proxy.Call(&s.persistLimit, "persistLimit"); err != nil {
		return errReadPersistLimit
	}

	s.registerProxyHandlers(s.proxyNetwork, *s.proxy.Address())

	var contracts []common.Address
	if contracts, err = s.storage.Contracts(); err != nil {
		return err
	}

	// update DB is proxy contract is not added as contract

	isProxyContractInDb := false
	for _, contract := range contracts {
		if contract == *s.proxy.Address() {
			isProxyContractInDb = true
			break
		}
	}
	if !isProxyContractInDb {
		if err = s.storage.AddContract(*s.proxy.Address()); err != nil {
			return err
		}
		contracts = append(contracts, *s.proxy.Address())
	}

	// -- register handlers for proxy addHash/addRemove

	for _, contract := range contracts {
		s.registerBasicHandlers(s.proxyNetwork, contract)
	}

	// -- register handlers for meta
	metadatahashes, err := s.storage.Metadatas()
	if err != nil {
		return err
	}
	for _, metadatahash := range metadatahashes {
		err = s.registerMetadataHandler(metadatahash)
		if err != nil {
			return err
		}
	}

	// -- start handlers

	for _, dispatcher := range s.dispatchers {
		dispatcher.Start()
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			log.Info("SERVE interrupt signal got. Finishing.")
			for _, dispatcher := range s.dispatchers {
				dispatcher.Stop()
			}
			break
		}
	}()
	for _, dispatcher := range s.dispatchers {
		dispatcher.Join()
	}

	return nil
}
