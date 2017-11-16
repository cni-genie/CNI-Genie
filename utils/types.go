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
package utils

import (
	"github.com/containernetworking/cni/pkg/types"
	c "github.com/google/cadvisor/info/v1"
	v1 "github.com/projectcalico/cni-plugin/utils"
	"net"
	"time"
)

type ContainerInfoGenie struct {
	// Historical statistics gathered from the container.
	Stats []ContainerStatsGenie `json:"stats,omitempty"`
}

type ContainerStatsGenie struct {
	// The time of this stat point.
	Timestamp time.Time      `json:"timestamp"`
	Network   c.NetworkStats `json:"network,omitempty"`
}

type InterfaceBandwidthUsage struct {
	IntName  string
	UpLink   uint64
	DownLink uint64
}

type AllInterfaces struct {
	Interfaces []c.InterfaceStats
}

// CNIArgs is a replica of skel.CmdArgs.
type CNIArgs struct {
	ContainerID string
	Netns       string
	IfName      string
	Args        string
	Path        string
	StdinData   []byte
}

// NetConf stores the common network config for Calico CNI plugin
type NetConf struct {
	CNIVersion string `json:"cniVersion"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	IPAM       struct {
		Name       string
		Type       string        `json:"type"`
		Subnet     string        `json:"subnet"`
		RangeStart string        `json:"rangeStart"`
		RangeEnd   string        `json:"rangeEnd"`
		Gateway    net.IP        `json:"gateway"`
		Routes     []types.Route `json:"routes"`
		AssignIpv4 *string       `json:"assign_ipv4"`
		AssignIpv6 *string       `json:"assign_ipv6"`
		IPv4Pools  []string      `json:"ipv4_pools,omitempty"`
		IPv6Pools  []string      `json:"ipv6_pools,omitempty"`
	} `json:"ipam,omitempty"`
	MTU            int                    `json:"mtu"`
	Hostname       string                 `json:"hostname"`
	DatastoreType  string                 `json:"datastore_type"`
	EtcdAuthority  string                 `json:"etcd_authority"`
	EtcdEndpoints  string                 `json:"etcd_endpoints"`
	LogLevel       string                 `json:"log_level"`
	Policy         v1.Policy              `json:"policy"`
	Kubernetes     v1.Kubernetes          `json:"kubernetes"`
	Args           v1.Args                `json:"args"`
	EtcdScheme     string                 `json:"etcd_scheme"`
	EtcdKeyFile    string                 `json:"etcd_key_file"`
	EtcdCertFile   string                 `json:"etcd_cert_file"`
	EtcdCaCertFile string                 `json:"etcd_ca_cert_file"`
	Delegate       map[string]interface{} `json:"delegate"`
	CalicoSubnet   string                 `json:"calico_subnet"`
	CanalSubnet    string                 `json:"canal_subnet"`
	WeaveSubnet    string                 `json:"weave_subnet"`

	Bridge           string `json:"bridge,omitempty"`
	IsDefaultGateway bool   `json:"isDefaultGateway,omitempty"`
	ForceAddress     bool   `json:"forceAddress,omitempty"`
	IpMasq           bool   `json:"ipMasq,omitempty"`
	HairpinMode      bool   `json:"hairpinMode,omitempty"`
	IsGateway        bool   `json:"isGateway,omitempty"`

	//added for romana
	RomanaRoot       string `json:"romana_root"`
	SegmentLabelName string `json:"segment_label_name"`

	//sriov cni specific
	IF0              string `json:"if0"`
	Vlan             int    `json:"vlan"`
}

// K8sArgs is the valid CNI_ARGS used for Kubernetes
type K8sArgs struct {
	types.CommonArgs
	IP                         net.IP
	K8S_POD_NAME               types.UnmarshallableString
	K8S_POD_NAMESPACE          types.UnmarshallableString
	K8S_POD_INFRA_CONTAINER_ID types.UnmarshallableString
}

// Temporary/alpha structures to support multiple ip addresses within Pod.

// A set of preferences that can be added to Pod as a json-serialized annotation.
// The preferences allow to express the number of ip addresses, ip addresses,
// their corresponding interfaces within the Pod.
type MultiIPPreferences struct {
	MultiEntry int64                           `json:"multi_entry,omitempty"`
	Ips        map[string]IPAddressPreferences `json:"ips,omitempty"`
}

type IPAddressPreferences struct {
	Ip        string `json:"ip,omitempty"`
	Interface string `json:"interface,omitempty"`
}
