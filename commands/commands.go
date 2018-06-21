package commands

import (
	"errors"
	"math/big"
	"os"

	"github.com/ipfsconsortium/gipc/service"
	sto "github.com/ipfsconsortium/gipc/storage"

	"github.com/spf13/cobra"
)

var (
	errInvalidParameters = errors.New("invalid parameters")
)

func Update(cmd *cobra.Command, args []string) {

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

}

// Serve command
func Serve(cmd *cobra.Command, args []string) {

	must(load(true))

	service.NewService(
		ethclients,
		1, nil,
		ipfs, storage,
	).Serve()

}
