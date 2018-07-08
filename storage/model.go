package storage

type HashEntry struct {
	Members  []string
	DataSize uint
}

type MemberEntry struct {
	HashCount uint
	DataSize  uint
}

type GlobalsEntry struct {
	CurrentQuota uint
}

type SavePointEntry struct {
	LastBlock    uint64
	LastTxIndex  uint
	LastLogIndex uint
}
