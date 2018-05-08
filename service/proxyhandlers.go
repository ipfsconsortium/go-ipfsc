package service

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func (s *Service) registerProxyHandlers(networkid uint64, address common.Address) {

	s.dispatchers[networkid].RegisterHandler(address, s.proxy.Abi(), "ContractAdded", s.handleContractAdded, nil)

}

func (s *Service) handleContractAdded(eventlog *types.Log, _ *ScanEventHandler) error {

	type ContractAdded struct {
		Member common.Address
		PubKey common.Address
		Ttl    *big.Int
	}

	var event ContractAdded
	err := s.proxy.Abi().Unpack(&event, "ContractAdded", eventlog.Data)
	if err != nil {
		return err
	}

	s.registerProxyHandlers(s.proxyNetwork, event.PubKey)

	return nil
}
