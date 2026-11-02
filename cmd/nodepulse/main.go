package main

import (
	"os"

	"github.com/HanphoneJan/nodepulse/internal/cli"
	"github.com/HanphoneJan/nodepulse/internal/version"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], version.Version))
}
