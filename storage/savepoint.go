package storage

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/rlp"
	log "github.com/sirupsen/logrus"
)

// Globals gets the globals from the db.
func (s *Storage) SavePoint(networkid uint64) (*SavePointEntry, error) {

	key, _, _ := s.savePointKV(networkid, nil)

	value, err := s.db.Get(key, nil)
	if err != nil {
		return nil, err
	}

	var entry SavePointEntry
	err = rlp.DecodeBytes(value, &entry)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// SetGlobals in the storage.
func (s *Storage) SetSavePoint(networkid uint64, sp *SavePointEntry) error {

	log.WithFields(log.Fields{
		"network":      networkid,
		"block/tx/log": fmt.Sprintf("%v/%v/%v", sp.LastBlock, sp.LastTxIndex, sp.LastLogIndex),
	}).Debug("DB savepoint")

	key, value, err := s.savePointKV(networkid, sp)
	if err != nil {
		return err
	}
	return s.db.Put(key, value, nil)
}

func (s *Storage) savePointKV(networkid uint64, entry *SavePointEntry) (key, value []byte, err error) {

	networkidbytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(networkidbytes, networkid)
	key = append([]byte(prefixSavePoint), networkidbytes...)

	if entry != nil {
		value, err = rlp.EncodeToBytes(entry)
		return key, value, err
	}

	return key, nil, nil
}
