package storage

import (
	common "github.com/ethereum/go-ethereum/common"
)

type HashContractEntry struct {
	Address common.Address
	Ttl     uint
}

type HashEntry struct {
	Contracts []HashContractEntry
	DataSize  uint
}

type ContractEntry struct {
	HashCount uint
}

type GlobalsEntry struct {
	CurrentQuota uint
}

type SavePointEntry struct {
	LastBlock    uint64
	LastTxIndex  uint
	LastLogIndex uint
}
