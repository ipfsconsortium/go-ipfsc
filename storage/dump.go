package storage

import (
	"bytes"
	"fmt"
	"io"

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

		}
	}

}
