package storage

import (
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func (s *Storage) AddMetadata(ipfsHash string) error {
	key := append([]byte(prefixMetadata), []byte(ipfsHash)...)

	log.WithField("hash", ipfsHash).Debug("DB add metadata")
	return s.db.Put(key, []byte{1}, nil)
}

func (s *Storage) RemoveMetadata(ipfsHash string) error {

	key := append([]byte(prefixMetadata), []byte(ipfsHash)...)
	return s.db.Delete(key, nil)

}

func (s *Storage) Metadatas() ([]string, error) {

	hashes := []string{}

	iter := s.db.NewIterator(util.BytesPrefix([]byte(prefixMetadata)), nil)
	defer iter.Release()

	for iter.Next() {
		key := iter.Key()
		ipfsHash := string(key[len(prefixMetadata):])
		hashes = append(hashes, ipfsHash)
	}
	return hashes, iter.Error()
}
