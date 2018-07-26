package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddHash(t *testing.T) {
	s := CreateTestDB(t)

	err := s.AddHash("h1", &HashEntry{
		DataSize: 1000,
		Links:    []string{"1", "2"},
		Mark:     true,
		Dirty:    false,
	})
	assert.Nil(t, err)

	h, err := s.Hash("h1")
	assert.Nil(t, err)
	assert.Equal(t, uint(1000), h.DataSize)
	assert.Equal(t, 2, len(h.Links))
	assert.Equal(t, "1", h.Links[0])
	assert.Equal(t, "2", h.Links[1])
	assert.Equal(t, true, h.Mark)
	assert.Equal(t, false, h.Dirty)
}

func TestGetUnexistentHash(t *testing.T) {
	s := CreateTestDB(t)

	h, err := s.Hash("h1")
	assert.Nil(t, err)
	assert.Nil(t, h)
}

func TestDeleteHash(t *testing.T) {
	s := CreateTestDB(t)

	err := s.AddHash("h1", &HashEntry{
		DataSize: 1000,
		Links:    []string{"1", "2"},
		Mark:     true,
		Dirty:    false,
	})
	assert.Nil(t, err)

	s.DeleteHash("h1")

	h, err := s.Hash("h1")
	assert.Nil(t, err)
	assert.Nil(t, h)
}

func TestUpdateHash(t *testing.T) {
	s := CreateTestDB(t)

	err := s.AddHash("h1", &HashEntry{
		DataSize: 1000,
		Links:    []string{"1", "2"},
		Mark:     true,
		Dirty:    false,
	})
	assert.Nil(t, err)
	err = s.UpdateHash("h1", &HashEntry{
		DataSize: 1001,
		Links:    []string{"3"},
		Mark:     false,
		Dirty:    true,
	})
	assert.Nil(t, err)

	h, err := s.Hash("h1")
	assert.Nil(t, err)
	assert.Equal(t, uint(1001), h.DataSize)
	assert.Equal(t, 1, len(h.Links))
	assert.Equal(t, "3", h.Links[0])
	assert.Equal(t, false, h.Mark)
	assert.Equal(t, true, h.Dirty)
}

func TestUpdateHashIter(t *testing.T) {
	s := CreateTestDB(t)

	assert.Nil(t, s.AddHash("h1", &HashEntry{
		DataSize: 1000, Links: []string{"1", "2"},
		Mark: true, Dirty: false,
	}))
	assert.Nil(t, s.AddHash("h2", &HashEntry{
		DataSize: 1001, Links: []string{"1", "2"},
		Mark: false, Dirty: false,
	}))
	assert.Nil(t, s.AddHash("h3", &HashEntry{
		DataSize: 1002, Links: []string{"1", "2"},
		Mark: true, Dirty: false,
	}))

	s.HashUpdateIter(func(hash string, entry *HashEntry) *HashEntry {
		if !entry.Mark {
			entry.Dirty = true
			return entry
		}
		return nil
	})

	h, err := s.Hash("h1")
	assert.Nil(t, err)
	assert.Equal(t, false, h.Dirty)

	h, err = s.Hash("h2")
	assert.Nil(t, err)
	assert.Equal(t, true, h.Dirty)

	h, err = s.Hash("h3")
	assert.Nil(t, err)
	assert.Equal(t, false, h.Dirty)
}
