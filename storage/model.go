package storage

type HashEntry struct {
	DataSize uint
	Links    []string
	Mark     bool
	Dirty    bool
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

type ResolvesEntry struct {
	Entries []string
}
