package main

import (
	"os"
	"runtime/debug"

	"github.com/nm10pterra/webalyze/internal/cli"
)

var appVersion = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, resolvedVersion()))
}

func resolvedVersion() string {
	if appVersion != "" && appVersion != "dev" {
		return appVersion
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return appVersion
}
