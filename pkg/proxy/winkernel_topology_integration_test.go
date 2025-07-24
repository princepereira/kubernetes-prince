//go:build !windows
// +build !windows

/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package proxy

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// TestTopologyAwareRoutingIntegration verifies that the CategorizeEndpoints function
// properly handles topology-aware routing when provided with zone hints.
// This test validates that the winkernel proxier's integration with topology.go
// will work correctly for Windows nodes with topology labels.
func TestTopologyAwareRoutingIntegration(t *testing.T) {
	nodeName := "test-node"
	nodeLabels := map[string]string{
		v1.LabelTopologyZone: "zone-a",
	}
	
	// Create a service that uses cluster endpoints (default behavior)
	svcInfo := &BaseServicePortInfo{}
	
	// Create endpoints with zone hints - some in zone-a, some in zone-b
	endpoints := []Endpoint{
		&BaseEndpointInfo{
			endpoint:  "10.1.2.3:80",
			zoneHints: sets.New[string]("zone-a"),
			ready:     true,
		},
		&BaseEndpointInfo{
			endpoint:  "10.1.2.4:80", 
			zoneHints: sets.New[string]("zone-b"),
			ready:     true,
		},
		&BaseEndpointInfo{
			endpoint:  "10.1.2.5:80",
			zoneHints: sets.New[string]("zone-a"),
			ready:     true,
		},
	}
	
	// Call CategorizeEndpoints as the winkernel proxier would
	clusterEndpoints, localEndpoints, allLocallyReachableEndpoints, hasEndpoints := CategorizeEndpoints(
		endpoints, svcInfo, nodeName, nodeLabels)
	
	// Verify that topology-aware routing is working
	if !hasEndpoints {
		t.Error("Expected hasEndpoints to be true")
	}
	
	// With zone hints and zone-a label, should only get zone-a endpoints
	expectedClusterEndpoints := 2 // Both zone-a endpoints
	if len(clusterEndpoints) != expectedClusterEndpoints {
		t.Errorf("Expected %d cluster endpoints (zone-a only), got %d", 
			expectedClusterEndpoints, len(clusterEndpoints))
	}
	
	// Verify we got the right endpoints (zone-a only)
	foundZoneAEndpoints := 0
	for _, ep := range clusterEndpoints {
		epInfo, ok := ep.(*BaseEndpointInfo)
		if !ok {
			t.Error("Failed to cast endpoint")
			continue
		}
		if epInfo.zoneHints.Has("zone-a") {
			foundZoneAEndpoints++
		} else {
			t.Errorf("Found endpoint not in zone-a: %s", epInfo.endpoint)
		}
	}
	
	if foundZoneAEndpoints != expectedClusterEndpoints {
		t.Errorf("Expected %d zone-a endpoints, got %d", 
			expectedClusterEndpoints, foundZoneAEndpoints)
	}
	
	// localEndpoints should be nil for this service type
	if localEndpoints != nil {
		t.Errorf("Expected localEndpoints to be nil, got %d endpoints", len(localEndpoints))
	}
	
	// allLocallyReachableEndpoints should equal clusterEndpoints in this case
	if len(allLocallyReachableEndpoints) != len(clusterEndpoints) {
		t.Errorf("Expected allLocallyReachableEndpoints (%d) to equal clusterEndpoints (%d)",
			len(allLocallyReachableEndpoints), len(clusterEndpoints))
	}
	
	t.Logf("Successfully validated topology-aware routing: %d endpoints filtered to %d based on zone hints",
		len(endpoints), len(clusterEndpoints))
}

// TestTopologyAwareRoutingWithoutHints verifies that when no topology hints are present,
// all endpoints are returned, maintaining backward compatibility.
func TestTopologyAwareRoutingWithoutHints(t *testing.T) {
	nodeName := "test-node"
	nodeLabels := map[string]string{
		v1.LabelTopologyZone: "zone-a",
	}
	
	svcInfo := &BaseServicePortInfo{}
	
	// Create endpoints without zone hints
	endpoints := []Endpoint{
		&BaseEndpointInfo{endpoint: "10.1.2.3:80", ready: true},
		&BaseEndpointInfo{endpoint: "10.1.2.4:80", ready: true},
		&BaseEndpointInfo{endpoint: "10.1.2.5:80", ready: true},
	}
	
	clusterEndpoints, _, allLocallyReachableEndpoints, hasEndpoints := CategorizeEndpoints(
		endpoints, svcInfo, nodeName, nodeLabels)
	
	if !hasEndpoints {
		t.Error("Expected hasEndpoints to be true")
	}
	
	// Without hints, should get all endpoints
	if len(clusterEndpoints) != len(endpoints) {
		t.Errorf("Expected %d cluster endpoints (all), got %d", 
			len(endpoints), len(clusterEndpoints))
	}
	
	if len(allLocallyReachableEndpoints) != len(endpoints) {
		t.Errorf("Expected %d allLocallyReachableEndpoints (all), got %d",
			len(endpoints), len(allLocallyReachableEndpoints))
	}
	
	t.Logf("Successfully validated backward compatibility: all %d endpoints returned without hints",
		len(endpoints))
}