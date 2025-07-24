# Topology Aware Routing Support for Windows Kernel Proxier

This implementation adds Topology Aware Routing support to the Windows kernel proxier (`winkernel`), enabling Windows worker nodes to benefit from zone-aware traffic routing.

## What Was Added

### 1. Topology Labels Storage
- Added `topologyLabels map[string]string` field to the `Proxier` struct to store node topology information

### 2. OnTopologyChange Method
- Implemented `OnTopologyChange(topologyLabels map[string]string)` method that:
  - Updates the proxier's topology labels when node labels change
  - Follows the same pattern as iptables and ipvs proxiers

### 3. Topology-Aware Endpoint Processing
- Modified `syncProxyRules()` to use `proxy.CategorizeEndpoints()` instead of direct endpoint iteration
- This enables the proxier to:
  - Route traffic preferentially to endpoints in the same zone
  - Fall back to endpoints in other zones when local zone endpoints are unavailable
  - Maintain backward compatibility when no topology hints are present

## How It Works

### Before (Without Topology Awareness)
```go
for _, epInfo := range proxier.endpointsMap[svcName] {
    // Process all endpoints equally
    ep, ok := epInfo.(*endpointInfo)
    // ... create load balancer rules for all endpoints
}
```

### After (With Topology Awareness)
```go
// Categorize endpoints using topology-aware logic
allEndpoints := proxier.endpointsMap[svcName]
clusterEndpoints, localEndpoints, allLocallyReachableEndpoints, hasEndpoints := 
    proxy.CategorizeEndpoints(allEndpoints, svcInfo, proxier.nodeName, proxier.topologyLabels)

// Use topology-aware endpoint selection
endpointsToProcess := allLocallyReachableEndpoints
for _, epInfo := range endpointsToProcess {
    // Process only topology-aware filtered endpoints
    ep, ok := epInfo.(*endpointInfo)
    // ... create load balancer rules for preferred endpoints
}
```

## Benefits

1. **Reduced Cross-Zone Traffic**: Traffic is routed to endpoints in the same zone when available
2. **Lower Latency**: Same-zone routing typically provides better performance
3. **Cost Reduction**: Reduced cross-zone data transfer costs in cloud environments
4. **Backward Compatibility**: Works seamlessly with existing services that don't use topology hints

## Example Usage

When a Windows node has the topology label:
```yaml
topology.kubernetes.io/zone: "us-west-2a"
```

And a service has endpoints with zone hints:
```yaml
apiVersion: v1
kind: Endpoints
metadata:
  name: my-service
  annotations:
    endpoints.kubernetes.io/zone-hints: |
      [{"name":"ep1","zone":"us-west-2a"},{"name":"ep2","zone":"us-west-2b"}]
```

The Windows kernel proxier will:
1. Prefer routing to endpoints in `us-west-2a` 
2. Fall back to `us-west-2b` endpoints only if no `us-west-2a` endpoints are available
3. Log the endpoint categorization for debugging

## Testing

The implementation includes:
- Unit tests for the `OnTopologyChange` functionality
- Integration tests validating topology-aware endpoint filtering
- All existing proxy tests continue to pass, ensuring no regressions

## Compatibility

- **Kubernetes Version**: Compatible with clusters running Kubernetes 1.23+ where Topology Aware Routing is beta
- **Windows Versions**: Works with all Windows versions supported by the winkernel proxier
- **Feature Gates**: Automatically uses topology hints when present, no additional feature gates required