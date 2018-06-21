package service

import (
	"errors"
	"math/big"
	"os"
	"os/signal"

	"github.com/ethereum/go-ethereum/ethclient"

	shell "github.com/adriamb/go-ipfs-api"
	eth "github.com/ipfsconsortium/gipc/eth"
	sto "github.com/ipfsconsortium/gipc/storage"
)

type Service struct {
	persistLimit *big.Int
	network      uint64
	ethclients   map[uint64]*ethclient.Client
	ens          *eth.Contract
	ipfs         *shell.Shell
	storage      *sto.Storage
}

var (
	errVerifySmartcontract = errors.New("cannot verify deployed smartcontract")
	errReadPersistLimit    = errors.New("error reading current persistLimit")
	errReachedPersistLimit = errors.New("persistlimit reached")
)

func NewService(ethclients map[uint64]*ethclient.Client, network uint64, ens *eth.Contract, ipfs *shell.Shell, storage *sto.Storage) *Service {

	return &Service{
		network:    network,
		ethclients: ethclients,
		ens:        ens,
		ipfs:       ipfs,
		storage:    storage,
	}
}

func (s *Service) Serve() error {

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
	}()
	return nil
}
