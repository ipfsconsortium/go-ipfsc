package service

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ipfsconsortium/gipc/storage"

	log "github.com/sirupsen/logrus"
)

type savePoint struct {
	storage *storage.Storage
}

func newSavePoint(storage *storage.Storage) *savePoint {
	return &savePoint{storage}
}

func (s *savePoint) Load() (lastBlock uint64, lastTxIndex, lastLogIndex uint, err error) {
	globals, err := s.storage.Globals()
	if err != nil {
		return 0, 0, 0, err
	}
	return globals.LastBlock, globals.LastTxIndex, globals.LastLogIndex, nil
}

func (s *savePoint) Save(logevent *types.Log) error {
	globals, err := s.storage.Globals()
	if err != nil {
		return err
	}

	globals.LastBlock = logevent.BlockNumber
	globals.LastTxIndex = logevent.TxIndex
	globals.LastLogIndex = logevent.Index

	log.WithFields(log.Fields{
		"block/tx/log": fmt.Sprintf("%v/%v/%v", globals.LastBlock, globals.LastTxIndex, globals.LastLogIndex),
	}).Debug("SAVEP save")

	return s.storage.SetGlobals(*globals)
}

func (s *savePoint) SkipTx(txid common.Hash) (bool, error) {
	return s.storage.SkipTx(txid)
}
