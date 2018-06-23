package storage

import (
	"github.com/ethereum/go-ethereum/rlp"
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// Contract return a new contract.
func (s *Storage) Member(member string) (*MemberEntry, error) {

	memberKey := append([]byte(prefixMember), []byte(member)[:]...)
	value, err := s.db.Get(memberKey, nil)
	if err != nil {
		return nil, err
	}

	var entry MemberEntry
	err = rlp.DecodeBytes(value, &entry)
	if err != nil {
		return nil, err
	}

	return &entry, nil
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

	key, _, _ := s.memberKV(&member, nil)
	_, err := s.db.Get(key, nil)

	if err == nil {
		return errContractAlreadyExists
	}

	entry := MemberEntry{
		HashCount: 0,
	}
	_, value, err := s.memberKV(nil, &entry)
	if err != nil {
		return err
	}

	log.WithField("contract", member).Debug("DB added contract")

	return s.db.Put(key, value, nil)
}

// RemoveMember from the storage..
func (s *Storage) RemoveMember(member string) error {
	key := append([]byte(prefixMember), []byte(member)[:]...)
	_, err := s.db.Get(key, nil)

	if err != nil {
		return errContractNotExists
	}

	log.WithField("member", member).Debug("DB removed member")
	return s.db.Delete(key, nil)
}

func (s *Storage) memberKV(name *string, member *MemberEntry) (key, value []byte, err error) {
	if member != nil {
		key = append([]byte(prefixMember), []byte(*name)[:]...)
	}
	if member != nil {
		value, err = rlp.EncodeToBytes(member)
	}
	return key, value, err
}
