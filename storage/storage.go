package storage

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	prefixHash    = "H"
	prefixMember  = "C"
	prefixGlobals = "G"
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

func (s *Storage) globalsKV(globals GlobalsEntry) (key, value []byte, err error) {

	key = []byte(prefixGlobals)

	value, err = rlp.EncodeToBytes(globals)
	return key, value, nil
}
