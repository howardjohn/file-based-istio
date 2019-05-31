package main

import (
	"fmt"
	"os"

	"istio.io/istio/pkg/adsc"
)

var (
	OUTDIR = os.Getenv("OUTDIR")
)

func main() {
	_ = os.MkdirAll(OUTDIR+"/rds", os.ModePerm)
	_ = os.MkdirAll(OUTDIR+"/eds", os.ModePerm)

	grpc := fmt.Sprintf("localhost:15010")
	adsc, err := adsc.Dial(grpc, "", &adsc.Config{
		IP: "10.60.11.42",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Waiting for updates")
	WriteXDSConfig(adsc)
}
