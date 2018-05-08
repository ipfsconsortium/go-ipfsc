package gometh

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// C is the package config
var C Config

// Config is the server configurtion
type Config struct {
	Keystore struct {
		Account string
		Path    string
		Passwd  string
	}

	Contracts struct {
		IPFSProxy struct {
			NetworkID uint64
			JSONURL   string
			Address   string
			Deploy    struct {
				Members      []string
				Required     uint
				PersistLimit uint64
			}
		}
	}

	DB struct {
		Path string
	}
	IPFS struct {
		APIURL string
	}

	Networks []struct {
		NetworkID  uint64
		RPCURL     string
		StartBlock uint64
	}
}

func (c *Config) Verify() error {

	if !common.IsHexAddress(c.Contracts.IPFSProxy.Address) {
		return fmt.Errorf("Bad Address %v", c.Contracts.IPFSProxy.Address)
	}
	return nil

}
