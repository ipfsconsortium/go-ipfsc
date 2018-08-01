package main

import (
	banner "github.com/CrowdSurge/banner"
	cmd "github.com/ipfsconsortium/go-ipfsc/cmd"
)

func main() {

	banner.Print("go-ipfsc")
	cmd.ExecuteCmd()
}
