package storage

import (
	"fmt"

	"github.com/ethereum/go-ethereum/rlp"
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	dberr "github.com/syndtr/goleveldb/leveldb/errors"
)

func (s *Storage) updateGlobalQuota(batch *leveldb.Batch, diff int) error {
	var err error
	var gkey, gvalue []byte
	var gentry *GlobalsEntry

	if gkey, gentry, err = s.globalsGet(); err != nil {
		return err
	}
	gentry.CurrentQuota = uint(int(gentry.CurrentQuota) + diff)
	log.WithField("quota", gentry.CurrentQuota).Debug("DB Quota updated")

	if gvalue, err = rlp.EncodeToBytes(gentry); err != nil {
		return err
	}

	batch.Put(gkey, gvalue)
	return nil
}

// AddHash to the storage.
func (s *Storage) AddHash(member string, hash string, size uint) error {

	var err error
	batch := new(leveldb.Batch)

	// --- update hash ---

	var hkey, hvalue []byte
	var hentry *HashEntry

	if hkey, hentry, err = s.hashGet(hash); err != nil {
		return err
	}

	if hentry != nil {

		// check if is the same information.
		if size != hentry.DataSize {
			return ErrInconsistentSize
		}

		// if element exists, return.
		for _, m := range hentry.Members {
			if m == member {
				return nil
			}
		}

		// add a new member to the hash.
		log.WithField("hash", hash).Debug("DB Adding member to hash.")
		hentry.Members = append(hentry.Members, member)

	} else {

		// hash entry does not exist, create new.
		log.WithField("hash", hash).Debug("DB Adding new hash.")

		hentry = &HashEntry{
			Members:  []string{member},
			DataSize: size,
		}

		// update globals.
		if err = s.updateGlobalQuota(batch, int(size)); err != nil {
			return err
		}

	}

	if hvalue, err = rlp.EncodeToBytes(hentry); err != nil {
		return err
	}

	batch.Put(hkey, hvalue)

	// --- update member ---

	var mkey, mvalue []byte
	var mentry *MemberEntry

	if mkey, mentry, err = s.memberGet(member); err != nil {
		return err
	}

	if mentry != nil {
		mentry.DataSize += size
		mentry.HashCount++
	} else {
		mentry = &MemberEntry{
			DataSize:  size,
			HashCount: 1,
		}
	}

	if mvalue, err = rlp.EncodeToBytes(mentry); err != nil {
		return err
	}

	batch.Put(mkey, mvalue)

	// --- update db ---

	return s.db.Write(batch, nil)
}

// RemoveHash from the storage.
func (s *Storage) RemoveHash(member string, hash string) error {

	var err error
	batch := new(leveldb.Batch)

	var hkey, hvalue []byte
	var hentry *HashEntry

	if hkey, hentry, err = s.hashGet(hash); err != nil {
		return err
	}
	if hentry == nil {
		return dberr.ErrNotFound
	}

	memberOffet := -1
	for i, m := range hentry.Members {
		if member == m {
			memberOffet = i
			break
		}
	}
	if memberOffet == -1 {
		return fmt.Errorf("member not in hash")
	}

	if len(hentry.Members) == 1 {

		// the only contract with this hash, delete all entry.
		log.WithField("hash", hash).Debug("DB Remove hash entry, hash removed")
		batch.Delete(hkey)

		// update globals.
		if err = s.updateGlobalQuota(batch, 0-int(hentry.DataSize)); err != nil {
			return err
		}

	} else {

		// remove the member in hash & save
		hentry.Members[memberOffet] = hentry.Members[len(hentry.Members)-1]
		hentry.Members = hentry.Members[:len(hentry.Members)-1]

		if hvalue, err = rlp.EncodeToBytes(hentry); err != nil {
			return err
		}

		log.WithField("Hash", hash).Debug("DB Remove hash entry, hash already in other contracts")
		batch.Put(hkey, hvalue)
	}

	// update the member
	var mkey, mvalue []byte
	var mentry *MemberEntry

	if mkey, mentry, err = s.memberGet(member); err != nil {
		return err
	}
	if mentry == nil {
		return dberr.ErrNotFound
	}

	mentry.DataSize -= hentry.DataSize
	mentry.HashCount--

	if mvalue, err = rlp.EncodeToBytes(mentry); err != nil {
		return err
	}

	batch.Put(mkey, mvalue)

	// --- update db ---

	return s.db.Write(batch, nil)

}

func (s *Storage) Hash(hash string) (*HashEntry, error) {

	var err error
	var hentry *HashEntry

	if _, hentry, err = s.hashGet(hash); err != nil {
		return nil, err
	}
	if hentry == nil {
		return nil, dberr.ErrNotFound
	}
	return hentry, nil

}

func (s *Storage) hashGet(hash string) ([]byte, *HashEntry, error) {
	hkey := append([]byte(prefixHash), []byte(hash)...)
	hvalue, err := s.db.Get(hkey, nil)
	if err == dberr.ErrNotFound {
		return hkey, nil, nil
	} else if err != nil {
		return nil, nil, nil
	}
	var hentry HashEntry
	err = rlp.DecodeBytes(hvalue, &hentry)
	return hkey, &hentry, err
}
