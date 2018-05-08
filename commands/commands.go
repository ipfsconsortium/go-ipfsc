package commands

import (
	"errors"
	"fmt"
	"math/big"
	"os"

	cfg "github.com/ipfsconsortium/gipc/config"
	"github.com/ipfsconsortium/gipc/service"
	sto "github.com/ipfsconsortium/gipc/storage"

	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	errInvalidParameters = errors.New("invalid parameters")
)

// HashAdd command
func AddHash(cmd *cobra.Command, args []string) {

	if len(args) != 2 {
		must(errInvalidParameters)
	}
	must(load(false))

	ttl := new(big.Int)
	ttl.SetString(args[1], 10)
	_, _, err := proxy.SendTransactionSync(
		big.NewInt(0), 0,
		"addHash", args[0], ttl,
	)
	must(err)
}

// HashRm command
func RemoveHash(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		must(errInvalidParameters)
	}
	must(load(false))
	_, _, err := proxy.SendTransactionSync(
		big.NewInt(0), 0,
		"removeHash", args[0],
	)
	must(err)
}

// MetaAdd command
func AddMetadataObject(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		must(errInvalidParameters)
	}
	must(load(false))

	_, _, err := proxy.SendTransactionSync(
		big.NewInt(0), 0,
		"addMetadataObject", args[0],
	)
	must(err)
}

// MetaRm command
func RemoveMetadataObject(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		must(errInvalidParameters)
	}
	must(load(false))
	_, _, err := proxy.SendTransactionSync(
		big.NewInt(0), 0,
		"removeMetadataObject", args[0],
	)
	must(err)
}

// AddMember command
func AddMember(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		must(errInvalidParameters)
	}
	must(load(false))
	_, _, err := proxy.SendTransactionSync(
		big.NewInt(0), 0,
		"addMember", common.BytesToAddress([]byte(args[0])),
	)
	must(err)
}

// RemoveMember command
func RemoveMember(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		must(errInvalidParameters)
	}
	must(load(false))
	_, _, err := proxy.SendTransactionSync(
		big.NewInt(0), 0,
		"removeMember", common.BytesToAddress([]byte(args[0])),
	)
	must(err)
}

// SetMemberRequirement command
func SetMemberRequirement(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		must(errInvalidParameters)
	}
	must(load(false))

	required := new(big.Int)
	required.SetString(args[0], 10)
	_, _, err := proxy.SendTransactionSync(
		big.NewInt(0), 0,
		"changeRequirement", required,
	)
	must(err)
}

// MemberInfo command
func MemberInfo(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		must(errInvalidParameters)
	}
	must(load(false))
	// TODO
}

// SetPersistLimit command
func SetPersistLimit(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		must(fmt.Errorf("setpersistlimit <limit>"))
	}
	must(load(false))
	limit := big.NewInt(0)

	if _, ok := limit.SetString(args[0], 10); !ok {
		must(fmt.Errorf("cannot parse limit parameter"))
	}

	_, _, err := proxy.SendTransactionSync(
		big.NewInt(0), 0,
		"setTotalPersistLimit", limit,
	)

	must(err)
}

// AddSkipTx command
func AddSkipTx(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		must(fmt.Errorf("skiptx <txhash>"))
	}
	must(load(true))
	must(storage.AddSkipTx(common.HexToHash(args[0])))
}

// AddSkipTx command
func DeployProxy(cmd *cobra.Command, args []string) {

	must(load(true))

	// -- deploy contract

	members := make([]common.Address, len(cfg.C.Contracts.IPFSProxy.Deploy.Members))
	for i, v := range cfg.C.Contracts.IPFSProxy.Deploy.Members {
		members[i] = common.HexToAddress(v)
	}

	_, _, err := proxy.DeploySync(
		members,
		big.NewInt(int64(cfg.C.Contracts.IPFSProxy.Deploy.Required)),
		big.NewInt(int64(cfg.C.Contracts.IPFSProxy.Deploy.PersistLimit)),
	)
	must(err)
	log.WithField("address", proxy.Address().Hex()).Info("Goic contract deployed")

	must(storage.AddContract(*proxy.Address()))

}

func DbRemoveMetadataObject(cmd *cobra.Command, args []string) {
	must(loadStorage())

	if len(args) != 1 {
		must(fmt.Errorf("pameter <ipfshash>"))
	}

	must(storage.RemoveMetadata(args[0]))
}

// DumpDb command
func DumpDb(cmd *cobra.Command, args []string) {

	must(loadStorage())

	storage.Dump(os.Stdout)
}

// InitDb command
func InitDb(cmd *cobra.Command, args []string) {

	must(loadStorage())

	storage.SetGlobals(sto.GlobalsEntry{
		CurrentQuota: 0,
	})

	for _, network := range cfg.C.Networks {
		storage.SetSavePoint(network.NetworkID, &sto.SavePointEntry{
			LastBlock:    network.StartBlock,
			LastTxIndex:  0,
			LastLogIndex: 0,
		})
	}
}

// Serve command
func Serve(cmd *cobra.Command, args []string) {

	must(load(true))

	service.NewService(
		ethclients,
		cfg.C.Contracts.IPFSProxy.NetworkID, proxy,
		ipfs, storage,
	).Serve()

}
