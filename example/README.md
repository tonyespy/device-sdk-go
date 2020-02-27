# device-simple

device-simple provide a simple example on how to develop Device Service using device-sdk-go.

## Protocol Driver

To make a functional Device Service, developers must implement [ProtocolDriver](../pkg/models/protocoldriver.go) interface. 
`ProtocolDriver` interface provides abstraction logic of how to interact with Device through specific protocol. See [simpledriver.go](driver/simpledriver.go) for example.

## Protocol Discovery

Some device protocols allow for devices to be discovered automatically.
A Device Service may include a capability for discovering devices and creating the corresponding Device objects within EdgeX.  

To enable device discovery, developers need to implement [ProtocolDiscovery](../pkg/models/protocoldiscovery.go) interface.
`ProtocolDiscovery` interface provides `Discovery` function to trigger protocol-specific discovery and return the details of of devices which it has found back to SDK. 
SDK will then filters these devices against a set of acceptance criteria and adds accepted devices into core-metadata.

The filter criteria for discovered devices are represented by Provision Watchers. A Provision Watcher contains the following fields:

`Identifiers`: A set of name-value pairs against which a new device's ProtocolProperties are matched  
`BlockingIdentifiers`: A further set of name-value pairs which are also matched against a new device's ProtocolProperties  
`Profile`: The name of a DeviceProfile which should be assigned to new devices which pass this ProvisionWatcher  
`AdminState`: The initial Administrative State for new devices which pass this ProvisionWatcher  
A candidate new device passes a ProvisionWatcher if all of the Identifiers match, and none of the BlockingIdentifiers.

Finally, A boolean configuration value `Device/Discovery/Enabled` defaults to false. If it is set true, and the DS implementation supports discovery, discovery is enabled.
Dynamic Device Discovery is triggered either on a internal timer(`Device/Discovery/Interval`) or by a call to the REST endpoint.

The following steps show how to trigger discovery on device-simple:
1. Set `Device/Discovery/Enabled` to true in [configuration file](cmd/device-simple/res/configuration.toml)
2. Post the [provided provisionwatcher](cmd/device-simple/res/provisionwatcher.json) into core-metadata endpoint: http://localhost:48081/api/v1/provisionwatcher
3. Trigger discovery by sending POST request to DS endpoint: http://localhost:49990/api/v1/discovery
4. `Simple-Device02` will be discovered and added to EdgeX.