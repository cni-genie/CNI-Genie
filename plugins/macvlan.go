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

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	// Macvlan specifies macvlan network
	Macvlan = "macvlan"
	// DefaultIpamForMacvlan specifies the default ipam type for macvlan
	DefaultIpamForMacvlan = "host-local"
	// DefaultSubnet denotes the default subnet to be used to assign ip from
	DefaultSubnet = "10.10.0.0/16"
	// DefaultMasterForMacvlan specifies the default interface for macvlan
	DefaultMasterForMacvlan = "eth0"
)

func GetMacvlanConfig() interface{} {
	master := DefaultMasterForMacvlan
	if nic, err := GetDefaultNic(); err == nil && nic != "" {
		master = nic
	} else {
		fmt.Fprintf(os.Stderr, "CNI Genie Could not get default nic for the host; [error: %v]. Using %s as master instead.", err, DefaultMasterForMacvlan)
	}

	macvlanObj := struct {
		Name   string      `json:"name"`
		Type   string      `json:"type"`
		Master string      `json:"master"`
		Ipam   interface{} `json:"ipam"`
	}{
		Name:   "macvlannet",
		Type:   Macvlan,
		Master: master,
		Ipam: struct {
			Type   string `json:"type"`
			Subnet string `json:"subnet"`
		}{
			Type:   DefaultIpamForMacvlan,
			Subnet: DefaultSubnet,
		},
	}

	return macvlanObj
}

func GetDefaultNic() (string, error) {
	cmd := `route | grep '^default' | grep -o '[^ ]*$'`
	output, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return "", err
	}

	nic := strings.TrimRight(fmt.Sprintf("%s", output), "\n")

	return nic, nil
}
