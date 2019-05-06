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
	"net"
	"time"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
	c "github.com/google/cadvisor/info/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Delimiter seperating plugin/network name from interfece name
	IfNameDelimiter = "@"
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

// PolicyConfig is a struct to hold policy config
type PolicyConfig struct {
	PolicyType              string `json:"type"`
	K8sAPIRoot              string `json:"k8s_api_root"`
	K8sAuthToken            string `json:"k8s_auth_token"`
	K8sClientCertificate    string `json:"k8s_client_certificate"`
	K8sClientKey            string `json:"k8s_client_key"`
	K8sCertificateAuthority string `json:"k8s_certificate_authority"`
}

// Kubernetes a K8s specific struct to hold config
type KubernetesConfig struct {
	K8sAPIRoot string `json:"k8s_api_root"`
	Kubeconfig string `json:"kubeconfig"`
}

// GenieConf describes cni-genie plugin configurations
type GenieConf struct {
	types.NetConf
	Kubernetes KubernetesConfig `json:"kubernetes"`
	Policy     PolicyConfig     `json:"policy"`
	LogLevel   string           `json:"log_level"`
	// CNI-Genie default plugin
	DefaultPlugin string `json:"default_plugin"`
	// Address to reach at cadvisor. By default, http://127.0.0.1:4194 is used as CAdvisor address
	CAdvisorAddr string `json:"cAdvisor_address"`
}

// K8sArgs is the valid CNI_ARGS used for Kubernetes
type K8sArgs struct {
	types.CommonArgs
	IP                         net.IP
	K8S_POD_NAME               types.UnmarshallableString
	K8S_POD_NAMESPACE          types.UnmarshallableString
	K8S_POD_INFRA_CONTAINER_ID types.UnmarshallableString
	K8S_ANNOT                  types.UnmarshallableString
}

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

// LogicalNetwork describes the details of logical network info for user pod
type LogicalNetwork struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              struct {
		PhysicalNet string `json:"physicalNet,omitempty"`
		SubSubnet   string `json:"sub_subnet,omitempty"`
		Plugin      string `json:"plugin,omitempty"`
	} `json:"spec"`
}

// LogicalNetworkList is a list of LogicalNetwork resource
type LogicalNetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []LogicalNetwork `json:"items"`
}

// PhysicalNetwork describes the details of physical network info for user pod
type PhysicalNetwork struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              struct {
		ReferNic     string `json:"refer_nic"`
		SharedStatus struct {
			Plugin          string `json:"plugin,omitempty"`
			Subnet          string `json:"subnet,omitempty"`
			DedicatedStatus bool   `json:"dedicatedNet"`
		} `json:"sharedStatus"`
	} `json:"spec"`
}

// PhysicalNetworkList is a list of PhysicalNetwork resource
type PhysicalNetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []PhysicalNetwork `json:"items"`
}

type ValidateResult func(types.Result, interface{}) error

// PluginInfo describes the details of plugin info for user pod
type PluginInfo struct {
	PluginName       string
	IfName           string
	Subnet           string
	Refer_nic        string
	Config           *libcni.NetworkConfigList
	OptionalArgs     map[string]string
	ValidationParams interface{}
	ValidateRes      ValidateResult
}
