package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddHash(t *testing.T) {
	s := CreateTestDB(t)

	err := s.AddHash("m1", "h1", 1000)
	assert.Nil(t, err)

	h, err := s.Hash("h1")
	assert.Nil(t, err)
	assert.Equal(t, uint(1000), h.DataSize)
	assert.Equal(t, 1, len(h.Members))
	assert.Equal(t, "m1", h.Members[0])

	m, err := s.Member("m1")
	assert.Nil(t, err)
	assert.Equal(t, uint(1000), m.DataSize)
	assert.Equal(t, uint(1), m.HashCount)

	g, err := s.Globals()
	assert.Nil(t, err)
	assert.Equal(t, uint(1000), g.CurrentQuota)
}

func TestSameMemberTwoHashes(t *testing.T) {
	var err error
	s := CreateTestDB(t)

	err = s.AddHash("m1", "h1", 1000)
	assert.Nil(t, err)
	err = s.AddHash("m1", "h2", 2000)
	assert.Nil(t, err)

	m, err := s.Member("m1")
	assert.Nil(t, err)
	assert.Equal(t, uint(3000), m.DataSize)
	assert.Equal(t, uint(2), m.HashCount)

	g, err := s.Globals()
	assert.Nil(t, err)
	assert.Equal(t, uint(3000), g.CurrentQuota)
}

func TestRepeatedHashInMember(t *testing.T) {
	var err error
	s := CreateTestDB(t)

	err = s.AddHash("m1", "h1", 1000)
	assert.Nil(t, err)
	err = s.AddHash("m1", "h1", 1000)
	assert.Nil(t, err)

	m, err := s.Member("m1")
	assert.Nil(t, err)
	assert.Equal(t, uint(1000), m.DataSize)
	assert.Equal(t, uint(1), m.HashCount)

	g, err := s.Globals()
	assert.Nil(t, err)
	assert.Equal(t, uint(1000), g.CurrentQuota)
}

func TestTwoMembersSameHash(t *testing.T) {
	var err error
	s := CreateTestDB(t)

	err = s.AddHash("m1", "h1", 1000)
	assert.Nil(t, err)
	err = s.AddHash("m2", "h1", 1000)
	assert.Nil(t, err)

	m, err := s.Member("m1")
	assert.Nil(t, err)
	assert.Equal(t, uint(1000), m.DataSize)
	assert.Equal(t, uint(1), m.HashCount)

	m, err = s.Member("m2")
	assert.Nil(t, err)
	assert.Equal(t, uint(1000), m.DataSize)
	assert.Equal(t, uint(1), m.HashCount)

	h, err := s.Hash("h1")
	assert.Nil(t, err)
	assert.Equal(t, uint(1000), h.DataSize)
	assert.Equal(t, 2, len(h.Members))
	assert.Equal(t, "m1", h.Members[0])
	assert.Equal(t, "m2", h.Members[1])

	g, err := s.Globals()
	assert.Nil(t, err)
	assert.Equal(t, uint(1000), g.CurrentQuota)
}
func TestSameHashSizeMismatch(t *testing.T) {
	var err error
	s := CreateTestDB(t)

	err = s.AddHash("m1", "h1", 1000)
	assert.Nil(t, err)
	err = s.AddHash("m2", "h1", 2000)
	assert.Equal(t, err, ErrInconsistentSize)
}

func TestRemoveSharedHash(t *testing.T) {
	var err error
	s := CreateTestDB(t)

	err = s.AddHash("m1", "h1", 1000)
	assert.Nil(t, err)
	err = s.AddHash("m2", "h1", 1000)
	assert.Nil(t, err)
	err = s.AddHash("m2", "h2", 2000)
	assert.Nil(t, err)
	err = s.RemoveHash("m1", "h1")
	assert.Nil(t, err)

	m, err := s.Member("m1")
	assert.Nil(t, err)
	assert.Equal(t, uint(0), m.DataSize)
	assert.Equal(t, uint(0), m.HashCount)

	m, err = s.Member("m2")
	assert.Nil(t, err)
	assert.Equal(t, uint(3000), m.DataSize)
	assert.Equal(t, uint(2), m.HashCount)

	h, err := s.Hash("h1")
	assert.Nil(t, err)
	assert.Equal(t, uint(1000), h.DataSize)
	assert.Equal(t, 1, len(h.Members))
	assert.Equal(t, "m2", h.Members[0])

	g, err := s.Globals()
	assert.Nil(t, err)
	assert.Equal(t, uint(3000), g.CurrentQuota)
}

func TestRemoveUniqueHash(t *testing.T) {
	var err error
	s := CreateTestDB(t)

	err = s.AddHash("m2", "h1", 1000)
	assert.Nil(t, err)
	err = s.AddHash("m2", "h2", 2000)
	assert.Nil(t, err)
	err = s.RemoveHash("m2", "h1")
	assert.Nil(t, err)

	m, err := s.Member("m2")
	assert.Nil(t, err)
	assert.Equal(t, uint(2000), m.DataSize)
	assert.Equal(t, uint(1), m.HashCount)

	g, err := s.Globals()
	assert.Nil(t, err)
	assert.Equal(t, uint(2000), g.CurrentQuota)
}
