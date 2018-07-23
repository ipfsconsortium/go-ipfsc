package storage

import (
	"github.com/ethereum/go-ethereum/rlp"
	log "github.com/sirupsen/logrus"
	dberr "github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// Contract return a new contract.
func (s *Storage) Member(member string) (*MemberEntry, error) {

	var err error
	var mentry *MemberEntry

	if _, mentry, err = s.memberGet(member); err != nil {
		return nil, err
	}
	if mentry == nil {
		return nil, dberr.ErrNotFound
	}
	return mentry, nil
}

// Contracts returns the list of current defined contracts.
func (s *Storage) Members() ([]string, error) {

	members := []string{}

	iter := s.db.NewIterator(util.BytesPrefix([]byte(prefixMember)), nil)
	defer iter.Release()

	for iter.Next() {
		key := iter.Key()
		member := string(key[len(prefixMember):])
		members = append(members, member)
	}
	return members, iter.Error()
}

// AddContract to the storage.
func (s *Storage) AddMember(member string) error {

	var err error
	var mkey, mvalue []byte
	var mentry *MemberEntry

	if mkey, mentry, err = s.memberGet(member); err != nil {
		return err
	}
	if mentry != nil {
		return ErrKeyAlreadyExists
	}

	mentry = &MemberEntry{
		HashCount: 0,
	}

	if mvalue, err = rlp.EncodeToBytes(mentry); err != nil {
		return err
	}

	log.WithField("contract", member).Debug("DB added contract")

	return s.db.Put(mkey, mvalue, nil)
}

// RemoveMember from the storage..
func (s *Storage) RemoveMember(member string) error {
	key := append([]byte(prefixMember), []byte(member)[:]...)
	_, err := s.db.Get(key, nil)
	if err == dberr.ErrNotFound {
		return ErrKeyNotExists
	}
	if err != nil {
		return err
	}

	log.WithField("member", member).Debug("DB removed member")
	return s.db.Delete(key, nil)
}

func (s *Storage) memberGet(member string) ([]byte, *MemberEntry, error) {
	mkey := append([]byte(prefixMember), []byte(member)...)
	mvalue, err := s.db.Get(mkey, nil)
	if err == dberr.ErrNotFound {
		return mkey, nil, nil
	} else if err != nil {
		return nil, nil, nil
	}
	var mentry MemberEntry
	err = rlp.DecodeBytes(mvalue, &mentry)
	return mkey, &mentry, err
}
