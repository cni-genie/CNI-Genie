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

package plugins

const (
	// SriovNet denotes sr-iov type network
	SriovNet = "sriov"
	// DefaultIpamForSriov denotes default ipam type for sr-iov
	DefaultIpamForSriov = "fixipam"
	// DefaultSubnetForSriov denotes the default subnet to be used
	DefaultSubnetForSriov = "10.55.206.0/26"
	// DefaultGatewayForSriov denotes the sefault gateway
	DefaultGatewayForSriov = "10.55.206.1"
)

func GetSriovConfig() interface{} {
	sriovObj := struct {
		Name   string      `json:"name"`
		Type   string      `json:"type"`
		Master string      `json:"master"`
		Ipam   interface{} `json:"ipam"`
	}{
		Name:   "sriovnet",
		Type:   SriovNet,
		Master: "eth0",
		Ipam: struct {
			Type    string              `json:"type"`
			Subnet  string              `json:"subnet"`
			Routes  []map[string]string `json:"routes"`
			Gateway string              `json:"gateway"`
		}{
			Type:   DefaultIpamForSriov,
			Subnet: DefaultSubnetForSriov,
			Routes: []map[string]string{
				{"dst": "0.0.0.0/0"},
			},
			Gateway: DefaultGatewayForSriov,
		},
	}

	return sriovObj
}
