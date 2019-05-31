package main

import (
	"bytes"
	"fmt"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	hcm_filter "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/envoyproxy/go-control-plane/pkg/util"
	"github.com/gogo/protobuf/types"
	"io/ioutil"
	"log"
	"os"
	"sigs.k8s.io/yaml"
	"strings"
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
	_ = os.MkdirAll(OUTDIR+"/rds", os.ModePerm)
	_ = os.MkdirAll(OUTDIR+"/eds", os.ModePerm)

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
	_, err = adsc.Wait("cds", 10*time.Second)
	_, err = adsc.Wait("lds", 10*time.Second)
	_, err = adsc.Wait("rds", 10*time.Second)
	_, err = adsc.Wait("eds", 10*time.Second)

	clusters := []*v2.Cluster{}
	for _, c := range adsc.Clusters {
		clusters = append(clusters, c)
	}
	for _, c := range adsc.EDSClusters {
		clusters = append(clusters, c)
	}
	write(clusterResponse(clusters), "cds.yaml")

	listeners := []*v2.Listener{}
	for _, l := range adsc.HTTPListeners {
		listeners = append(listeners, l)
	}
	for _, l := range adsc.TCPListeners {
		listeners = append(listeners, l)
	}
	write(listenerResponse(listeners), "lds.yaml")

	for route, r := range adsc.Routes {
		write(routesResponse([]*v2.RouteConfiguration{r}), fmt.Sprintf("rds/%s.yaml", sanitizeName(route)))
	}

	for endpoint, e := range adsc.EDS {
		write(endpointsResponse([]*v2.ClusterLoadAssignment{e}), fmt.Sprintf("eds/%s.yaml", sanitizeName(endpoint)))
	}
}

func endpointsResponse(response []*v2.ClusterLoadAssignment) *v2.DiscoveryResponse {
	out := &v2.DiscoveryResponse{
		TypeUrl:     pilotv2.EndpointType,
		VersionInfo: "0",
	}

	for _, c := range response {
		cc, _ := types.MarshalAny(c)
		out.Resources = append(out.Resources, *cc)
	}

	return out
}

func write(r *v2.DiscoveryResponse, out string) {
	if OUTDIR == "" {
		fmt.Println(string(MarshallYaml(r)))
	} else {
		if err := ioutil.WriteFile(OUTDIR+"/"+out, MarshallYaml(r), 0777); err != nil {
			panic(err)
		}
	}
}

func clusterResponse(response []*v2.Cluster) *v2.DiscoveryResponse {
	out := &v2.DiscoveryResponse{
		TypeUrl:     pilotv2.ClusterType,
		VersionInfo: "0",
	}

	sanitizeClusterAds(response)

	for _, c := range response {
		cc, _ := types.MarshalAny(c)
		out.Resources = append(out.Resources, *cc)
	}

	return out
}

func listenerResponse(response []*v2.Listener) *v2.DiscoveryResponse {
	out := &v2.DiscoveryResponse{
		TypeUrl:     pilotv2.ListenerType,
		VersionInfo: "0",
	}

	sanitizeListenerAds(response)

	for _, c := range response {
		cc, _ := types.MarshalAny(c)
		out.Resources = append(out.Resources, *cc)
	}

	return out
}

func routesResponse(response []*v2.RouteConfiguration) *v2.DiscoveryResponse {
	out := &v2.DiscoveryResponse{
		TypeUrl:     pilotv2.RouteType,
		VersionInfo: "0",
	}

	for _, c := range response {
		cc, _ := types.MarshalAny(c)
		out.Resources = append(out.Resources, *cc)
	}

	return out
}

func sanitizeClusterAds(response []*v2.Cluster) {
	for _, r := range response {
		if r.EdsClusterConfig == nil {
			continue
		}
		path := fmt.Sprintf("/etc/config/eds/%s.yaml", sanitizeName(r.Name))
		r.EdsClusterConfig.EdsConfig = &core.ConfigSource{
			ConfigSourceSpecifier: &core.ConfigSource_Path{Path: path},
		}
	}
}

func sanitizeListenerAds(response []*v2.Listener) {
	for _, c := range response {
		for _, fc := range c.FilterChains {
			for _, f := range fc.Filters {
				switch f.Name {
				case "envoy.http_connection_manager":
					routeName := f.GetConfig().Fields["rds"].GetStructValue().GetFields()["route_config_name"].GetStringValue()
					path := fmt.Sprintf("/etc/config/rds/%s.yaml", sanitizeName(routeName))
					rdsFileConfig, _ := util.MessageToStruct(&hcm_filter.Rds{
						RouteConfigName: routeName,
						ConfigSource: core.ConfigSource{
							ConfigSourceSpecifier: &core.ConfigSource_Path{Path: path},
						},
					})
					f.GetConfig().Fields["rds"] = &types.Value{
						Kind: &types.Value_StructValue{StructValue: rdsFileConfig},
					}
				default:
				}
			}
		}
	}
}

func sanitizeName(name string) string {
	return strings.ReplaceAll(name, "|", "_.")
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
