package main

import "github.com/camisul/nixmgr/cmd"

const (
	Version = "v0.2.0"
)

func main() {
	cmd.SetVersion(Version)
	cmd.Execute()
}
