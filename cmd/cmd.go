package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"
	"istio.io/istio/pkg/adsc"

	"github.com/howardjohn/file-based-istio/client"
)

var (
	outdir       = ""
	pilotAddress = "localhost:15010"
	nodeIp       = "0.0.0.0"
	namespace    = "default"
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&outdir, "outdir", "o", outdir, "directory to output to")
	rootCmd.PersistentFlags().StringVarP(&pilotAddress, "pilot-address", "p", pilotAddress, "address to pilot")
	rootCmd.PersistentFlags().StringVarP(&nodeIp, "pod-ip", "i", nodeIp, "ip address of pod to simulate (-ojsonpath='{.status.podIP}')")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", namespace, "namespace to use for the pod")
}

var rootCmd = &cobra.Command{
	Use:   "file-envoy",
	Short: "Convert Pilot XDS config to a file based XDS config",
	RunE: func(cmd *cobra.Command, args []string) error {
		if outdir != "" {
			_ = os.MkdirAll(path.Join(outdir, "rds"), os.ModePerm)
			_ = os.MkdirAll(path.Join(outdir, "eds"), os.ModePerm)
		}

		adsc, err := adsc.Dial(pilotAddress, "", &adsc.Config{
			IP:        nodeIp,
			Namespace: namespace,
			Meta: map[string]string{
				"CONFIG_NAMESPACE": namespace,
			},
		})
		if err != nil {
			ErrorExit("Failed to connect to pilot: %v", err)
		}
		fmt.Println("Waiting for updates")
		return client.WriteXDSConfig(adsc, outdir)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func ErrorExit(msg string, a ...interface{}) {
	fmt.Printf(msg, a)
	fmt.Println()
	os.Exit(1)
}
