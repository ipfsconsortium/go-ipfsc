package storage

import (
	"github.com/ethereum/go-ethereum/rlp"
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
)

// AddHash to the storage.
func (s *Storage) AddHash(member string, hash string, ttl, size uint) error {

	batch := new(leveldb.Batch)

	// update hash

	hkey, hvalue, update, err := s.addHashKV(member, hash, size)
	if err != nil {
		return err
	}
	if hkey == nil {
		return nil
	}

	batch.Put(hkey, hvalue)

	if update {

		// update member

		memberdata, err := s.Member(member)

		if err != nil {
			return err
		}

		memberdata.HashCount++

		ckey, cvalue, err := s.memberKV(&member, memberdata)
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
func (s *Storage) RemoveHash(member string, hash string) (bool, error) {

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
	var memberOffet int = -1
	for i, m := range entry.Members {
		if member == m {
			memberOffet = i
			break
		}
	}
	if memberOffet == -1 {
		// contract is not in this hash, return
		return false, nil
	}
	if len(entry.Members) == 1 {
		// the only contract with this hash, delete all entry, return
		log.WithField("Hash", hash).Debug("DB Remove hash entry, hash removed")

		return true, s.db.Delete(key, nil)
	}

	// remove the contract in hash & save
	entry.Members[memberOffet] = entry.Members[len(entry.Members)-1]
	entry.Members = entry.Members[:len(entry.Members)-1]

	value, err = rlp.EncodeToBytes(entry)
	if err != nil {
		return false, err
	}

	log.WithField("Hash", hash).Debug("DB Remove hash entry, hash already in other contracts")
	return false, s.db.Put(key, value, nil)
}

func (s *Storage) addHashKV(member string, hash string, size uint) (key, value []byte, update bool, err error) {

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

		exists := false
		for _, m := range entry.Members {
			if m == member {
				exists = true
				break
			}
		}
		if !exists {
			// add a new member
			log.WithField("hash", hash).Debug("DB Adding contract to hash.")
			entry.Members = append(entry.Members, member)
			update = true
		}
	} else {

		// new entry
		log.WithField("hash", hash).Debug("DB Adding new hash.")

		entry = HashEntry{
			Members:  []string{member},
			DataSize: size,
		}
		update = true

	}

	value, err = rlp.EncodeToBytes(entry)
	return key, value, update, err
}
