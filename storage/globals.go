package storage

import (
	"github.com/ethereum/go-ethereum/rlp"
	dberr "github.com/syndtr/goleveldb/leveldb/errors"
)

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

	var err error

	gkey := []byte(prefixGlobals)
	var gvalue []byte

	if gvalue, err = rlp.EncodeToBytes(globals); err != nil {
		return err
	}

	return s.db.Put(gkey, gvalue, nil)
}

// Globals gets the globals from the db.
func (s *Storage) globalsGet() ([]byte, *GlobalsEntry, error) {
	gkey := []byte(prefixGlobals)
	gvalue, err := s.db.Get(gkey, nil)
	if err == dberr.ErrNotFound {
		return gkey, nil, nil
	} else if err != nil {
		return nil, nil, nil
	}
	var gentry GlobalsEntry
	err = rlp.DecodeBytes(gvalue, &gentry)
	return gkey, &gentry, err
}
