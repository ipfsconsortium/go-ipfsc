package storage

import (
	common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// Contract return a new contract.
func (s *Storage) Contract(contract common.Address) (*ContractEntry, error) {

	contractKey := append([]byte(prefixContract), contract[:]...)
	value, err := s.db.Get(contractKey, nil)
	if err != nil {
		return nil, err
	}

	var entry ContractEntry
	err = rlp.DecodeBytes(value, &entry)
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

// Contracts returns the list of current defined contracts.
func (s *Storage) Contracts() ([]common.Address, error) {

	contracts := []common.Address{}

	iter := s.db.NewIterator(util.BytesPrefix([]byte(prefixContract)), nil)
	defer iter.Release()

	for iter.Next() {
		key := iter.Key()
		var contract common.Address
		copy(contract[:], key[len(prefixContract):])
		contracts = append(contracts, contract)
	}
	return contracts, iter.Error()
}

// AddContract to the storage.
func (s *Storage) AddContract(address common.Address) error {

	key, _, _ := s.contractKV(&address, nil)
	_, err := s.db.Get(key, nil)

	if err == nil {
		return errContractAlreadyExists
	}

	entry := ContractEntry{
		HashCount: 0,
	}
	_, value, err := s.contractKV(nil, &entry)
	if err != nil {
		return err
	}

	log.WithField("contract", address.Hex()).Debug("DB added contract")

	return s.db.Put(key, value, nil)
}

// RemoveContract AddContract to the storage..
func (s *Storage) RemoveContract(contract common.Address) error {
	key := append([]byte(prefixContract), contract[:]...)
	_, err := s.db.Get(key, nil)

	if err != nil {
		return errContractNotExists
	}

	log.WithField("contract", contract.Hex()).Debug("DB removed contract")
	return s.db.Delete(key, nil)
}

func (s *Storage) contractKV(address *common.Address, contract *ContractEntry) (key, value []byte, err error) {
	if address != nil {
		key = append([]byte(prefixContract), address[:]...)
	}
	if contract != nil {
		value, err = rlp.EncodeToBytes(contract)
	}
	return key, value, err
}
