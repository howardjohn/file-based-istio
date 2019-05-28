package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	adminapi "github.com/envoyproxy/go-control-plane/envoy/admin/v2alpha"
	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"google.golang.org/grpc"

	ads "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/ghodss/yaml"
	"istio.io/istio/istioctl/pkg/util/configdump"
)

const (
	typePrefix = "type.googleapis.com/envoy.api.v2."

	// Constants used for XDS

	// ClusterType is used for cluster discovery. Typically first request received
	clusterType = typePrefix + "Cluster"
	// EndpointType is used for EDS and ADS endpoint discovery. Typically second request.
	endpointType = typePrefix + "ClusterLoadAssignment"
	// ListenerType is sent after clusters and endpoints.
	listenerType = typePrefix + "Listener"
	// RouteType is sent after listeners.
	routeType = typePrefix + "RouteConfiguration"
)

func alt() {
	url := fmt.Sprintf("http://localhost:8080/debug/config_dump?proxyID=%s", "svc-0-0-0-5d8c5dd9fb-trlnl.test")
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	msg := &adminapi.ConfigDump{}
	if err := jsonpb.Unmarshal(resp.Body, msg); err != nil {
		panic(err)
	}

	wrapper := configdump.Wrapper{ConfigDump: msg}
	fmt.Println(wrapper.GetBootstrapConfigDump())
	fmt.Println(wrapper.GetClusterConfigDump())
	cluster, _ := wrapper.GetClusterConfigDump()
	o, _ := MarshallJson(cluster)
	y, _ := yaml.JSONToYAML(o)
	fmt.Println(string(y))

}

// Config for the ADS connection.
type Config struct {
	// Namespace defaults to 'default'
	Namespace string

	// Workload defaults to 'test'
	Workload string

	// Meta includes additional metadata for the node
	Meta map[string]string

	// NodeType defaults to sidecar. "ingress" and "router" are also supported.
	NodeType string

	// IP is currently the primary key used to locate inbound configs. It is sent by client,
	// must match a known endpoint IP. Tests can use a ServiceEntry to register fake IPs.
	IP string
}

func (a *ADSC) node() *core.Node {
	n := &core.Node{
		Id: a.nodeID,
	}
	if a.Metadata == nil {
		n.Metadata = &types.Struct{
			Fields: map[string]*types.Value{
				"ISTIO_PROXY_VERSION": {Kind: &types.Value_StringValue{StringValue: "1.0"}},
			}}
	} else {
		f := map[string]*types.Value{}

		for k, v := range a.Metadata {
			f[k] = &types.Value{Kind: &types.Value_StringValue{StringValue: v}}
		}
		n.Metadata = &types.Struct{
			Fields: f,
		}
	}
	return n
}

type ADSC struct {
	conn *grpc.ClientConn

	// Stream is the GRPC connection stream, allowing direct GRPC send operations.
	// Set after Dial is called.
	stream ads.AggregatedDiscoveryService_StreamAggregatedResourcesClient

	url string

	// NodeID is the node identity sent to Pilot.
	nodeID string

	// Updates includes the type of the last update received from the server.
	Updates chan *xdsapi.DiscoveryResponse
	// Metadata has the node metadata to send to pilot.
	// If nil, the defaults will be used.
	Metadata map[string]string
}

func getPrivateIPIfAvailable() net.IP {
	addrs, _ := net.InterfaceAddrs()
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		default:
			continue
		}
		if !ip.IsLoopback() {
			return ip
		}
	}
	return net.IPv4zero
}
func Dial(url string, opts *Config) (*ADSC, error) {
	adsc := &ADSC{
		url:     url,
		Updates: make(chan *xdsapi.DiscoveryResponse, 100),
	}
	if opts.Namespace == "" {
		opts.Namespace = "default"
	}
	if opts.NodeType == "" {
		opts.NodeType = "sidecar"
	}
	if opts.IP == "" {
		opts.IP = getPrivateIPIfAvailable().String()
	}
	if opts.Workload == "" {
		opts.Workload = "test-1"
	}
	adsc.Metadata = opts.Meta

	adsc.nodeID = fmt.Sprintf("%s~%s~%s.%s~%s.svc.cluster.local", opts.NodeType, opts.IP,
		opts.Workload, opts.Namespace, opts.Namespace)

	err := adsc.Run()
	return adsc, err
}

// Run will run the ADS client.
func (a *ADSC) Run() error {

	// TODO: pass version info, nonce properly
	var err error
	a.conn, err = grpc.Dial(a.url, grpc.WithInsecure())
	if err != nil {
		return err
	}

	xds := ads.NewAggregatedDiscoveryServiceClient(a.conn)
	edsstr, err := xds.StreamAggregatedResources(context.Background())
	if err != nil {
		return err
	}
	a.stream = edsstr
	go a.handleRecv()
	_ = a.stream.Send(&xdsapi.DiscoveryRequest{
		ResponseNonce: time.Now().String(),
		Node:          a.node(),
		TypeUrl:       clusterType,
	})
	return nil
}

func (a *ADSC) handleRecv() {
	for {
		fmt.Println("recv")
		msg, err := a.stream.Recv()
		if err != nil {
			log.Println("Connection closed ", err, a.nodeID)
			return
		}
		a.Updates <- msg
		//listeners := []*xdsapi.Listener{}
		//clusters := []*xdsapi.Cluster{}
		//routes := []*xdsapi.RouteConfiguration{}
		//eds := []*xdsapi.ClusterLoadAssignment{}
		//for _, rsc := range msg.Resources { // Any
		//	valBytes := rsc.Value
		//	if rsc.TypeUrl == listenerType {
		//		ll := &xdsapi.Listener{}
		//		_ = proto.Unmarshal(valBytes, ll)
		//		listeners = append(listeners, ll)
		//	} else if rsc.TypeUrl == clusterType {
		//		ll := &xdsapi.Cluster{}
		//		_ = proto.Unmarshal(valBytes, ll)
		//		clusters = append(clusters, ll)
		//	} else if rsc.TypeUrl == endpointType {
		//		ll := &xdsapi.ClusterLoadAssignment{}
		//		_ = proto.Unmarshal(valBytes, ll)
		//		eds = append(eds, ll)
		//	} else if rsc.TypeUrl == routeType {
		//		ll := &xdsapi.RouteConfiguration{}
		//		_ = proto.Unmarshal(valBytes, ll)
		//		routes = append(routes, ll)
		//	}
	}
}

func main() {

	grpc := fmt.Sprintf("localhost:15010")
	adsc, err := Dial(grpc, &Config{
		IP:        "10.11.0.1",
		Namespace: "envoy",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Waiting for updates")
	for {
		select {
		case r := <-adsc.Updates:
			fmt.Println(r)
		case <-time.After(time.Second * 10):
			fmt.Println("Done")
			return
		}
	}
}

func MarshallJson(w proto.Message) ([]byte, error) {
	buffer := &bytes.Buffer{}
	err := (&jsonpb.Marshaler{}).Marshal(buffer, w)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
