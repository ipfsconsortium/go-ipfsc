package storage

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"

	common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	prefixHash     = "H"
	prefixContract = "C"
	prefixGlobals  = "G"
	prefixSkipTx   = "S"
)

var (
	errContractAlreadyExists = errors.New("contract already exists in db")
	errContractNotExists     = errors.New("contract does not exists in db")
	errInconsistentSize      = errors.New("inconsistent db and IPFS datasize")
)

type Storage struct {
	mutex *sync.Mutex
	db    *leveldb.DB
}

type HashContractEntry struct {
	Address common.Address
	Ttl     uint
}

type HashEntry struct {
	Contracts []HashContractEntry
	DataSize  uint
}

type ContractEntry struct {
	HashCount uint
}

type GlobalsEntry struct {
	CurrentQuota uint

	LastBlock    uint64
	LastTxIndex  uint
	LastLogIndex uint
}

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

func (s *Storage) addHashKV(contract common.Address, hash string, ttl, size uint) (key, value []byte, update bool, err error) {

	key = append([]byte(prefixHash), []byte(hash)...)

	value, err = s.db.Get(key, nil)
	var entry HashEntry

	if err == nil {

		err := rlp.DecodeBytes(value, &entry)
		if err != nil {
			return nil, nil, false, err
		}
		if size != entry.DataSize {
			return nil, nil, false, errInconsistentSize
		}
		var contractEntry *HashContractEntry
		for i, v := range entry.Contracts {
			if v.Address == contract {
				contractEntry = &entry.Contracts[i]
				break
			}
		}
		if contractEntry != nil {

			if contractEntry.Ttl == ttl {
				// exactly the same ttl for hash contract, return
				log.WithField("hash", hash).Debug("DB Entry already exists.")
				return nil, nil, false, err
			}

			// update TTL for existing hash in contract
			log.WithField("hash", hash).Debug("DB Updating TTL of hash")
			contractEntry.Ttl = ttl
			update = false

		} else {

			// add a new contract
			log.WithField("hash", hash).Debug("DB Adding contract to hash.")
			entry.Contracts = append(entry.Contracts, HashContractEntry{
				Address: contract,
				Ttl:     ttl,
			})
			update = true

		}
	} else {

		// new entry
		log.WithField("hash", hash).Debug("DB Adding new hash.")

		entry = HashEntry{
			Contracts: []HashContractEntry{
				HashContractEntry{
					Address: contract,
					Ttl:     ttl,
				},
			},
			DataSize: size,
		}
		update = true

	}

	value, err = rlp.EncodeToBytes(entry)
	return key, value, update, err
}

func (s *Storage) AddHash(address common.Address, hash string, ttl, size uint) error {

	batch := new(leveldb.Batch)

	// update hash

	hkey, hvalue, update, err := s.addHashKV(address, hash, ttl, size)
	if err != nil {
		return err
	}
	if hkey == nil {
		return nil
	}

	batch.Put(hkey, hvalue)

	if update {

		// update contract

		contract, err := s.Contract(address)

		if err != nil {
			return err
		}

		contract.HashCount++

		ckey, cvalue, err := s.contractKV(&address, contract)
		if err != nil {
			return err
		}
		batch.Put(ckey, cvalue)

		// update globals

		globals, err := s.Globals()
		if err != nil {
			return err
		}
		globals.CurrentQuota += size

		log.WithField("quota", globals.CurrentQuota).Debug("DB Quota updated")

		gkey, gvalue, err := s.globalsKV(*globals)
		batch.Put(gkey, gvalue)
	}

	return s.db.Write(batch, nil)
}

func (s *Storage) RemoveHash(contract common.Address, hash string) (bool, error) {

	key := append([]byte(prefixHash), []byte(hash)...)
	var entry HashEntry

	var err error

	value, err := s.db.Get(key, nil)
	if err != nil {
		log.WithField("hash", hash).Debug("DB Hash does not exist")
		// does not exist, return
		return false, err
	}

	err = rlp.DecodeBytes(value, &entry)
	if err != nil {
		return false, err
	}
	var contractOffet int = -1
	for i, v := range entry.Contracts {
		if v.Address == contract {
			contractOffet = i
			break
		}
	}
	if contractOffet == -1 {
		// contract is not in this hash, return
		return false, nil
	}
	if len(entry.Contracts) == 1 {
		// the only contract with this hash, delete all entry, return
		log.WithField("Hash", hash).Debug("DB Remove hash entry, hash removed")

		return true, s.db.Delete(key, nil)
	}

	// remove the contract in hash & save
	entry.Contracts[contractOffet] = entry.Contracts[len(entry.Contracts)-1]
	entry.Contracts = entry.Contracts[:len(entry.Contracts)-1]

	value, err = rlp.EncodeToBytes(entry)
	if err != nil {
		return false, err
	}

	log.WithField("Hash", hash).Debug("DB Remove hash entry, hash already in other contracts")
	return false, s.db.Put(key, value, nil)
}

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

func (s *Storage) globalsKV(globals GlobalsEntry) (key, value []byte, err error) {

	key = []byte(prefixGlobals)

	value, err = rlp.EncodeToBytes(globals)
	return key, value, nil
}

func (s *Storage) SetGlobals(globals GlobalsEntry) error {

	key, value, err := s.globalsKV(globals)
	if err != nil {
		return err
	}
	return s.db.Put(key, value, nil)
}

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

func (s *Storage) contractKV(address *common.Address, contract *ContractEntry) (key, value []byte, err error) {
	if address != nil {
		key = append([]byte(prefixContract), address[:]...)
	}
	if contract != nil {
		value, err = rlp.EncodeToBytes(contract)
	}
	return key, value, err
}

func (s *Storage) RemoveContract(contract common.Address) error {
	key := append([]byte(prefixContract), contract[:]...)
	_, err := s.db.Get(key, nil)

	if err != nil {
		return errContractNotExists
	}

	log.WithField("contract", contract.Hex()).Debug("DB removed contract")
	return s.db.Delete(key, nil)
}

func (s *Storage) AddSkipTx(txid common.Hash) error {
	key := append([]byte(prefixSkipTx), txid[:]...)

	log.WithField("tx", txid.Hex()).Debug("DB add skiptx")
	return s.db.Put(key, []byte{1}, nil)
}

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

func isPrefix(key []byte, prefix string) bool {
	return bytes.Equal([]byte(prefix), key[:len(prefix)])
}

func (s *Storage) Dump(w io.Writer) {

	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		key := iter.Key()
		value := iter.Value()

		switch {
		case isPrefix(key, prefixHash):

			w.Write([]byte(fmt.Sprintf("HASH %v", string(key[len(prefixHash):]))))

			var entry HashEntry
			err := rlp.DecodeBytes(value, &entry)
			if err != nil {
				w.Write([]byte("| *READ ERROR"))
				break
			}

			w.Write([]byte(fmt.Sprintf(" size=%v\n", entry.DataSize)))
			for _, contract := range entry.Contracts {
				w.Write([]byte(fmt.Sprintf("| CONTRACT %v\n", contract.Address.Hex())))
			}

		case isPrefix(key, prefixContract):

			var contract common.Address
			copy(contract[:], key[len(prefixContract):])

			w.Write([]byte(fmt.Sprintf("CONTRACT %v", contract.Hex())))

			var entry ContractEntry
			err := rlp.DecodeBytes(value, &entry)
			if err != nil {
				w.Write([]byte("| *READ ERROR"))
				break
			}

			w.Write([]byte(fmt.Sprintf("\n| hashcount=%v\n", entry.HashCount)))

		case isPrefix(key, prefixGlobals):

			w.Write([]byte("GLOBALS "))

			var entry GlobalsEntry
			err := rlp.DecodeBytes(value, &entry)
			if err != nil {
				w.Write([]byte("| *READ ERROR"))
				break
			}

			w.Write([]byte(fmt.Sprintf(
				"\n| currentQuota=%v\n| lastBlock=%v\n| lastTxIndex=%v\n| lastLogIndex=%v\n",
				entry.CurrentQuota, entry.LastBlock, entry.LastTxIndex, entry.LastLogIndex,
			)))

		case isPrefix(key, prefixSkipTx):

			var txid common.Hash
			copy(txid[:], key[len(prefixSkipTx):])

			w.Write([]byte(fmt.Sprintf("SKIPTX %v\n",
				txid.Hex(),
			)))

		}
	}

}
