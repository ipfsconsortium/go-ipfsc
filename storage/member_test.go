package storage

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func CreateTestDB(t *testing.T) *Storage {
	tmp, err := ioutil.TempDir("", "dbtest")
	assert.Nil(t, err)
	s, err := New(tmp)
	assert.Nil(t, err)
	err = s.SetGlobals(GlobalsEntry{
		CurrentQuota: 0,
	})
	assert.Nil(t, err)
	return s
}

func TestAddMember(t *testing.T) {
	s := CreateTestDB(t)

	err := s.AddMember("1")
	assert.Nil(t, err)
	members, err := s.Members()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(members))
	assert.Equal(t, "1", members[0])

	err = s.AddMember("2")
	assert.Nil(t, err)
	members, err = s.Members()
	assert.Nil(t, err)
	assert.Equal(t, 2, len(members))
	assert.Equal(t, "1", members[0])
	assert.Equal(t, "2", members[1])
}

func TestAddExistingMember(t *testing.T) {
	s := CreateTestDB(t)

	err := s.AddMember("1")
	assert.Nil(t, err)
	err = s.AddMember("1")
	assert.Equal(t, ErrKeyAlreadyExists, err)
}

func TestRemoveMember(t *testing.T) {
	s := CreateTestDB(t)

	err := s.AddMember("1")
	assert.Nil(t, err)
	err = s.AddMember("2")
	assert.Nil(t, err)

	err = s.RemoveMember("1")
	assert.Nil(t, err)
	members, err := s.Members()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(members))
	assert.Equal(t, "2", members[0])
}

func TestRemoveUnexistingMember(t *testing.T) {
	s := CreateTestDB(t)

	err := s.RemoveMember("1")
	assert.Equal(t, ErrKeyNotExists, err)
}

func TestGetMember(t *testing.T) {
	s := CreateTestDB(t)

	err := s.AddMember("1")
	assert.Nil(t, err)

	m, err := s.Member("1")
	assert.Nil(t, err)

	assert.Equal(t, uint(0), m.HashCount)
}
