package gometh

// C is the package config
var C Config

// Config is the server configurtion
type Config struct {
	Keystore struct {
		Account string
		Path    string
		Passwd  string
	}

	EnsNames struct {
		Network uint64
		Local   string
		Remotes []string
	}

	DB struct {
		Path string
	}

	IPFS struct {
		APIURL string
	}

	Networks map[uint64]struct {
		MaxGasPrice uint64
		EnsRoot     string
		RPCURL      string
	}
}
