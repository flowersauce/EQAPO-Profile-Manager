//go:build windows

// eqm manages AutoEQ profiles in a local Equalizer APO installation.
package main

import (
	"os"

	"github.com/flowersauce/EQAPO-Profile-Manager/internal/cli"
)

// version can be supplied with -ldflags "-X main.version=1.0.0".
var version = "1.0.0"

func main() { os.Exit(cli.Main(os.Args[1:], version)) }
