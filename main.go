package main

import (
	"fmt"

	banner "github.com/CrowdSurge/banner"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	cmd "github.com/ipfsconsortium/gipc/cmd"
	eth "github.com/ipfsconsortium/gipc/eth"
)

func main() {

	var acc accounts.Account
	web3, err := eth.NewWeb3ClientWithURL(
		"https://mainnet.infura.io/pA160itSfDvztBALCyV2",
		nil,
		acc,
	)
	if err != nil {
		panic(err)
	}

	addr := common.HexToAddress("0x314159265dd8dbb310642f98f50c066173c1259b")
	ens, err := eth.NewENSClient(web3, &addr)
	if err != nil {
		panic(err)
	}
	text, err := ens.GetText("consortium.dappnode.eth")
	if err != nil {
		panic(err)
	}

	fmt.Println("====== ", text, "========")

	banner.Print("gipc")
	fmt.Println("IPFS Consortium go implementation.")
	cmd.ExecuteCmd()
}
