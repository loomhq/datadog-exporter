package main

import (
	"os"

	"github.com/loomhq/datadog-exporter/cmd"
)

// version is populated at build time by goreleaser.
var version = "dev"

func main() {
	os.Exit(cmd.Execute(version, os.Args[1:], os.Stdout, os.Stderr))
}
