package eth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNameHash(t *testing.T) {
	assert.Equal(t, "0xde9b09fd7c5f901e23a3f19fecc54828e9c848539801e86591bd9801b019f84f", NameHash("foo.eth").Hex())
	assert.Equal(t, "0xe9c4d1125fa0b2d4dfb1853f542f31acb2b1fca6c0b69a0e714041a6aeb642a4", NameHash("codecontext.eth").Hex())
}
