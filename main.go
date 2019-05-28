package main

import (
	"fmt"

	"github.com/envoyproxy"
	adminapi "github.com/envoyproxy/go-control-plane/envoy/admin/v2alpha"
	//xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	//ads "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	//
	//"istio.io/istio/pilot/pkg/model"
	//"istio.io/istio/pilot/pkg/networking/util"
	//istiolog "istio.io/pkg/log"
)

func main() {
	dynamicActiveClusters := []adminapi.ClustersConfigDump_DynamicCluster{}
	clusters, err := s.generateRawClusters(conn.modelNode, s.globalPushContext())
	if err != nil {
		return nil, err
	}
	for _, cs := range clusters {
		dynamicActiveClusters = append(dynamicActiveClusters, adminapi.ClustersConfigDump_DynamicCluster{Cluster: cs})
	}
	clustersAny, err := types.MarshalAny(&adminapi.ClustersConfigDump{
		VersionInfo:           versionInfo(),
		DynamicActiveClusters: dynamicActiveClusters,
	})
	if err != nil {
		return nil, err
	}

	dynamicActiveListeners := []adminapi.ListenersConfigDump_DynamicListener{}
	listeners, err := s.generateRawListeners(conn, s.globalPushContext())
	if err != nil {
		return nil, err
	}
	for _, cs := range listeners {
		dynamicActiveListeners = append(dynamicActiveListeners, adminapi.ListenersConfigDump_DynamicListener{Listener: cs})
	}
	listenersAny, err := types.MarshalAny(&adminapi.ListenersConfigDump{
		VersionInfo:            versionInfo(),
		DynamicActiveListeners: dynamicActiveListeners,
	})
	if err != nil {
		return nil, err
	}

	routes, err := s.generateRawRoutes(conn, s.globalPushContext())
	if err != nil {
		return nil, err
	}
	routeConfigAny, _ := types.MarshalAny(&adminapi.RoutesConfigDump{})
	if len(routes) > 0 {
		dynamicRouteConfig := []adminapi.RoutesConfigDump_DynamicRouteConfig{}
		for _, rs := range routes {
			dynamicRouteConfig = append(dynamicRouteConfig, adminapi.RoutesConfigDump_DynamicRouteConfig{RouteConfig: rs})
		}
		routeConfigAny, err = types.MarshalAny(&adminapi.RoutesConfigDump{DynamicRouteConfigs: dynamicRouteConfig})
		if err != nil {
			return nil, err
		}
	}

	bootstrapAny, _ := types.MarshalAny(&adminapi.BootstrapConfigDump{})
	// The config dump must have all configs with order specified in
	// https://www.envoyproxy.io/docs/envoy/latest/api-v2/admin/v2alpha/config_dump.proto
	configDump := &adminapi.ConfigDump{Configs: []types.Any{*bootstrapAny, *clustersAny, *listenersAny, *routeConfigAny}}
	return configDump, nil
	fmt.Println("Running")
}


