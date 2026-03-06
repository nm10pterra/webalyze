package main

import (
	"fmt"
	"io"
)

const cliBanner = "" +
	"                __          __               \n" +
	" _      _____  / /_  ____ _/ /_  ______  ___ \n" +
	"| | /| / / _ \\/ __ \\/ __ `/ / / / /_  / / _ \\\n" +
	"| |/ |/ /  __/ /_/ / /_/ / / /_/ / / /_/  __/\n" +
	"|__/|__/\\___/_.___/\\__,_/_/\\__, / /___/\\___/\n" +
	"                          /____/              \n"

func printBanner(w io.Writer, version string) {
	fmt.Fprint(w, cliBanner)
	fmt.Fprintf(w, "              @nm10pt | version: %s\n\n", version)
}
