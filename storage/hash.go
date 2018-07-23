package storage

import (
	"github.com/ethereum/go-ethereum/rlp"
	dberr "github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func (s *Storage) AddHash(hash string, hentry *HashEntry) error {

	hkey := append([]byte(prefixHash), []byte(hash)...)
	hvalue, err := s.db.Get(hkey, nil)
	if err != dberr.ErrNotFound {
		return err
	}

	if hvalue, err = rlp.EncodeToBytes(hentry); err != nil {
		return err
	}

	if s.db.Put(hkey, hvalue, nil) != err {
		return err
	}
	return nil
}

func (s *Storage) DeleteHash(hash string) {

	hkey := append([]byte(prefixHash), []byte(hash)...)
	s.db.Delete(hkey, nil)
}

func (s *Storage) UpdateHash(hash string, hentry *HashEntry) error {

	var err error

	hkey := append([]byte(prefixHash), []byte(hash)...)

	var hvalue []byte
	if hvalue, err = rlp.EncodeToBytes(hentry); err != nil {
		return err
	}

	return s.db.Put(hkey, hvalue, nil)
}

type UpdateFunc func(hash string, entry *HashEntry) *HashEntry

func (s *Storage) HashUpdateIter(uf UpdateFunc) error {
	var err error

	iter := s.db.NewIterator(util.BytesPrefix([]byte(prefixHash)), nil)
	defer iter.Release()

	for iter.Next() {
		hkey := iter.Key()
		hash := string(hkey[len(prefixHash):])
		var hentry HashEntry
		if err = rlp.DecodeBytes(iter.Value(), &hentry); err != nil {
			return err
		}

		if updated := uf(hash, &hentry); updated != nil {
			var hvalue []byte
			if hvalue, err = rlp.EncodeToBytes(updated); err != nil {
				return err
			}
			if err := s.db.Put(hkey, hvalue, nil); err != nil {
				return err
			}
		}
	}
	return iter.Error()
}
func (s *Storage) Hash(hash string) (*HashEntry, error) {

	hkey := append([]byte(prefixHash), []byte(hash)...)
	hvalue, err := s.db.Get(hkey, nil)
	if err == dberr.ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var hentry HashEntry
	err = rlp.DecodeBytes(hvalue, &hentry)
	return &hentry, err

}
