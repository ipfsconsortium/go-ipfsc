package storage

import (
	"errors"
	"sync"

	common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	prefixHash      = "H"
	prefixContract  = "C"
	prefixGlobals   = "G"
	prefixSkipTx    = "S"
	prefixMetadata  = "M"
	prefixSavePoint = "P"
)

var (
	errContractAlreadyExists = errors.New("contract already exists in db")
	errContractNotExists     = errors.New("contract does not exists in db")
	errInconsistentSize      = errors.New("inconsistent db and IPFS datasize")
)

// Storage manages the application state
type Storage struct {
	mutex *sync.Mutex
	db    *leveldb.DB
}

// New creates a new storage path.
func New(path string) (*Storage, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	return &Storage{
		db:    db,
		mutex: &sync.Mutex{},
	}, nil
}

// Globals gets the globals from the db.
func (s *Storage) Globals() (*GlobalsEntry, error) {

	key := []byte(prefixGlobals)
	value, err := s.db.Get(key, nil)

	var entry GlobalsEntry
	err = rlp.DecodeBytes(value, &entry)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// SetGlobals in the storage.
func (s *Storage) SetGlobals(globals GlobalsEntry) error {

	key, value, err := s.globalsKV(globals)
	if err != nil {
		return err
	}
	return s.db.Put(key, value, nil)
}

// AddSkipTx marks a transaction to not be processed.
func (s *Storage) AddSkipTx(txid common.Hash) error {
	key := append([]byte(prefixSkipTx), txid[:]...)

	log.WithField("tx", txid.Hex()).Debug("DB add skiptx")
	return s.db.Put(key, []byte{1}, nil)
}

// SkipTx returns true if a transaction should't be processed.
func (s *Storage) SkipTx(txid common.Hash) (bool, error) {
	key := append([]byte(prefixSkipTx), txid[:]...)
	_, err := s.db.Get(key, nil)
	if err == nil {
		return true, nil
	} else if err == leveldb.ErrNotFound {
		return false, nil
	}

	return false, err
}

func (s *Storage) globalsKV(globals GlobalsEntry) (key, value []byte, err error) {

	key = []byte(prefixGlobals)

	value, err = rlp.EncodeToBytes(globals)
	return key, value, nil
}
