package main

import (
	"os"

	"istio.io/file-envoy/cmd"
)

var (
	OUTDIR = os.Getenv("OUTDIR")
)

func main() {
	cmd.Execute()
}
