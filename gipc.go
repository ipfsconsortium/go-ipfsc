package main

import (
	"fmt"

	banner "github.com/CrowdSurge/banner"
	cmd "github.com/ipfsconsortium/go-ipfsc/cmd"
)

func main() {

	banner.Print("gipc")
	fmt.Println("IPFS Consortium go implementation.")
	cmd.ExecuteCmd()
}
