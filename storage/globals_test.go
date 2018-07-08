package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetGlobals(t *testing.T) {
	s := CreateTestDB(t)

	err := s.SetGlobals(GlobalsEntry{
		CurrentQuota: 1313,
	})
	assert.Nil(t, err)

	g, err := s.Globals()
	assert.Nil(t, err)
	assert.Equal(t, uint(1313), g.CurrentQuota)
}
