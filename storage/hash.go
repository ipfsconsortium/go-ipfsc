package storage

import (
	common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
)

// AddHash to the storage.
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

// RemoveHash from the storage.
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