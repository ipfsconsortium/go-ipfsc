package commands

import (
	"errors"
	"os"

	"github.com/ipfsconsortium/gipc/service"
	sto "github.com/ipfsconsortium/gipc/storage"

	"github.com/spf13/cobra"
)

var (
	errInvalidParameters = errors.New("invalid parameters")
)

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

func Update(cmd *cobra.Command, args []string) {
}

// Serve command
func Serve(cmd *cobra.Command, args []string) {

	must(load(true))

	service.NewService(
		ens,
		ipfs, storage,
	).Serve()

}
