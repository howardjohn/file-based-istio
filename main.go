package main

import (
	"bytes"
	"fmt"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/gogo/protobuf/types"
	"io/ioutil"
	"log"
	"os"
	"sigs.k8s.io/yaml"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"

	pilotv2 "istio.io/istio/pilot/pkg/proxy/envoy/v2"
	"istio.io/istio/pkg/adsc"
)

var (
	OUTDIR = os.Getenv("OUTDIR")
)

func main() {
	log.SetFlags(0)
	grpc := fmt.Sprintf("localhost:15010")
	adsc, err := adsc.Dial(grpc, "", &adsc.Config{
		IP:        "10.11.0.1",
		Namespace: "envoy",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Waiting for updates")
	adsc.Watch()
	_, err = adsc.Wait("eds", 10*time.Second)
	clusters := []*v2.Cluster{}
	for _, c := range adsc.Clusters {
		clusters = append(clusters, c)
	}
	for _, c := range adsc.EDSClusters {
		clusters = append(clusters, c)
	}

	write(clusterResponse(clusters), "cds.yaml")
}

func write(r *v2.DiscoveryResponse, out string) {
	if OUTDIR == "" {
		fmt.Println(string(MarshallYaml(r)))
	} else {
		if err := ioutil.WriteFile(OUTDIR + "/" + out, MarshallYaml(r), 0777); err != nil {
			panic(err)
		}
	}
}
func clusterResponse(response []*v2.Cluster) *v2.DiscoveryResponse {
	out := &v2.DiscoveryResponse{
		TypeUrl:     pilotv2.ClusterType,
		VersionInfo: "0",
	}

	for _, c := range response {
		cc, _ := types.MarshalAny(c)
		out.Resources = append(out.Resources, *cc)
	}

	return out
}
func MarshallJson(w proto.Message) []byte {
	buffer := &bytes.Buffer{}
	err := (&jsonpb.Marshaler{}).Marshal(buffer, w)
	if err != nil {
		return []byte{}
	}
	return buffer.Bytes()
}

func MarshallYaml(w proto.Message) []byte {
	b, _ := yaml.JSONToYAML([]byte(MarshallJson(w)))
	return b
}
