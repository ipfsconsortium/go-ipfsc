package storage

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

func isPrefix(key []byte, prefix string) bool {
	return bytes.Equal([]byte(prefix), key[:len(prefix)])
}

// Dump the database content.
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
			for _, member := range entry.Members {
				w.Write([]byte(fmt.Sprintf("| MEMBER %v\n", member)))
			}

		case isPrefix(key, prefixMember):

			member := string(key[len(prefixMember):])

			w.Write([]byte(fmt.Sprintf("MEMBER %v", member)))

			var entry MemberEntry
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
				"\n| CurrentQuota=%v\n",
				entry.CurrentQuota,
			)))

		case isPrefix(key, prefixSavePoint):

			networkidbytes := make([]byte, 8)

			copy(networkidbytes[:], key[len(prefixSavePoint):])
			networkid := binary.LittleEndian.Uint64(networkidbytes)

			w.Write([]byte(fmt.Sprintf("SAVEPOINT %v", networkid)))
			var entry SavePointEntry
			err := rlp.DecodeBytes(value, &entry)
			if err != nil {
				w.Write([]byte("| *READ ERROR"))
				break
			}

			w.Write([]byte(fmt.Sprintf(
				"\n| lastBlock=%v\n| lastTxIndex=%v\n| lastLogIndex=%v\n",
				entry.LastBlock, entry.LastTxIndex, entry.LastLogIndex,
			)))

		case isPrefix(key, prefixSkipTx):

			var txid common.Hash
			copy(txid[:], key[len(prefixSkipTx):])

			w.Write([]byte(fmt.Sprintf("SKIPTX %v\n",
				txid.Hex(),
			)))

		case isPrefix(key, prefixMetadata):

			ipfsHash := string(key[len(prefixMetadata):])

			w.Write([]byte(fmt.Sprintf("METADATA %v\n",
				ipfsHash,
			)))

		}
	}

}
