package service

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ipfsconsortium/gipc/storage"
)

type savePoint struct {
	storage   *storage.Storage
	networkid uint64
}

func newSavePoint(storage *storage.Storage, networkid uint64) *savePoint {
	return &savePoint{storage, networkid}
}

func (s *savePoint) Load() (lastBlock uint64, lastTxIndex, lastLogIndex uint, err error) {
	sp, err := s.storage.SavePoint(s.networkid)
	if err != nil {
		return 0, 0, 0, err
	}
	return sp.LastBlock, sp.LastTxIndex, sp.LastLogIndex, nil
}

func (s *savePoint) Save(logevent *types.Log) error {

	sp, err := s.storage.SavePoint(s.networkid)
	if err != nil {
		return err
	}

	sp = &storage.SavePointEntry{
		LastBlock:    logevent.BlockNumber,
		LastTxIndex:  logevent.TxIndex,
		LastLogIndex: logevent.Index,
	}

	return s.storage.SetSavePoint(s.networkid, sp)
}

func (s *savePoint) SkipTx(txid common.Hash) (bool, error) {
	return s.storage.SkipTx(txid)
}
