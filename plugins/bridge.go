//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package utils maintains various type definitions used by CNI-Genie.
// It has for now a multi-purpose function to sort a map based on values.

package plugins

const (
	// BridgeNet specifies bridge type network
	BridgeNet = "bridge"
	// DefaultIpamForBridge specifies default ipam type for bridge
	DefaultIpamForBridge = "host-local"
	// DefaultSubnetForBridge specifies default subnet for bridge
	DefaultSubnetForBridge = "10.10.0.1/16"
)

func GetBridgeConfig() interface{} {
	bridgeObj := struct {
		Name string      `json:"name"`
		Type string      `json:"type"`
		Ipam interface{} `json:"ipam"`
	}{
		Name: "mybridgenet",
		Type: BridgeNet,
		Ipam: struct {
			Type   string `json:"type"`
			Subnet string `json:"subnet"`
		}{
			Type:   DefaultIpamForBridge,
			Subnet: DefaultSubnetForBridge,
		},
	}

	return bridgeObj
}
